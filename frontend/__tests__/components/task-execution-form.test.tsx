import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import '@testing-library/jest-dom'
import { TaskExecutionForm } from '@/components/task-execution-form'
import { useProjects, useCreateJob } from '@/hooks/use-job-api'

// Mock the hooks
jest.mock('@/hooks/use-job-api')

const mockUseProjects = useProjects as jest.MockedFunction<typeof useProjects>
const mockUseCreateJob = useCreateJob as jest.MockedFunction<typeof useCreateJob>

describe('TaskExecutionForm', () => {
  const mockCreateJob = jest.fn()
  const mockOnJobCreated = jest.fn()

  beforeEach(() => {
    jest.clearAllMocks()
    
    mockUseProjects.mockReturnValue({
      projects: [
        { id: '1', name: 'Project 1', path: '/path/to/project1' },
        { id: '2', name: 'Project 2', path: '/path/to/project2' }
      ],
      loading: false,
      error: null,
      refetch: jest.fn()
    } as any)

    mockUseCreateJob.mockReturnValue({
      createJob: mockCreateJob,
      loading: false,
      error: null
    } as any)
  })

  it('renders form with all fields', () => {
    render(<TaskExecutionForm onJobCreated={mockOnJobCreated} />)
    
    expect(screen.getByText('タスク実行')).toBeInTheDocument()
    expect(screen.getByLabelText('プロジェクト')).toBeInTheDocument()
    expect(screen.getByLabelText('コマンド')).toBeInTheDocument()
    expect(screen.getByLabelText('YOLOモード')).toBeInTheDocument()
    expect(screen.getByLabelText('実行タイミング')).toBeInTheDocument()
  })

  it('submits form with immediate execution', async () => {
    mockCreateJob.mockResolvedValue({ id: 'job-123' })
    
    render(<TaskExecutionForm onJobCreated={mockOnJobCreated} />)
    
    // Select project
    fireEvent.mouseDown(screen.getByLabelText('プロジェクト'))
    fireEvent.click(screen.getByText('Project 1'))
    
    // Enter command
    fireEvent.change(screen.getByLabelText('コマンド'), {
      target: { value: 'test command' }
    })
    
    // Submit form
    fireEvent.click(screen.getByText('タスクを実行'))
    
    await waitFor(() => {
      expect(mockCreateJob).toHaveBeenCalledWith({
        project_id: '1',
        command: 'test command',
        yolo_mode: false,
        schedule_type: 'immediate'
      })
      expect(mockOnJobCreated).toHaveBeenCalled()
    })
  })

  it('submits form with delayed execution', async () => {
    mockCreateJob.mockResolvedValue({ id: 'job-124' })
    
    render(<TaskExecutionForm onJobCreated={mockOnJobCreated} />)
    
    // Select project
    fireEvent.mouseDown(screen.getByLabelText('プロジェクト'))
    fireEvent.click(screen.getByText('Project 1'))
    
    // Enter command
    fireEvent.change(screen.getByLabelText('コマンド'), {
      target: { value: 'delayed command' }
    })
    
    // Select schedule type
    fireEvent.mouseDown(screen.getByLabelText('実行タイミング'))
    fireEvent.click(screen.getByText('N時間後に実行'))
    
    // Slider should appear
    expect(screen.getByText(/実行まで:/)).toBeInTheDocument()
    
    // Submit form
    fireEvent.click(screen.getByText('タスクを実行'))
    
    await waitFor(() => {
      expect(mockCreateJob).toHaveBeenCalledWith({
        project_id: '1',
        command: 'delayed command',
        yolo_mode: false,
        schedule_type: 'delayed',
        schedule_params: {
          delay_hours: 1
        }
      })
    })
  })

  it('validates scheduled execution requires date and time', async () => {
    render(<TaskExecutionForm onJobCreated={mockOnJobCreated} />)
    
    // Select project
    fireEvent.mouseDown(screen.getByLabelText('プロジェクト'))
    fireEvent.click(screen.getByText('Project 1'))
    
    // Enter command
    fireEvent.change(screen.getByLabelText('コマンド'), {
      target: { value: 'scheduled command' }
    })
    
    // Select scheduled type
    fireEvent.mouseDown(screen.getByLabelText('実行タイミング'))
    fireEvent.click(screen.getByText('日時を指定'))
    
    // Submit button should be disabled without date/time
    const submitButton = screen.getByText('タスクを実行')
    expect(submitButton).toBeDisabled()
    
    // Fill date and time
    fireEvent.change(screen.getByLabelText('実行日'), {
      target: { value: '2025-12-01' }
    })
    fireEvent.change(screen.getByLabelText('実行時刻'), {
      target: { value: '10:30' }
    })
    
    // Submit button should be enabled
    expect(submitButton).not.toBeDisabled()
  })

  it('displays error message on failure', async () => {
    const errorHook = {
      createJob: mockCreateJob,
      loading: false,
      error: 'Failed to create job'
    }
    mockUseCreateJob.mockReturnValue(errorHook as any)
    
    render(<TaskExecutionForm onJobCreated={mockOnJobCreated} />)
    
    expect(screen.getByText('Failed to create job')).toBeInTheDocument()
  })

  it('shows loading state during submission', async () => {
    const loadingHook = {
      createJob: mockCreateJob,
      loading: true,
      error: null
    }
    mockUseCreateJob.mockReturnValue(loadingHook as any)
    
    render(<TaskExecutionForm onJobCreated={mockOnJobCreated} />)
    
    expect(screen.getByText('実行中...')).toBeInTheDocument()
  })
})