// Next.js API Route Proxy
// This keeps the API key secure on the server side

import type { NextApiRequest, NextApiResponse } from 'next';

const BACKEND_URL = process.env.BACKEND_API_URL || 'http://localhost:6060/api';
const API_KEY = process.env.API_KEY; // Server-side only (no NEXT_PUBLIC_)

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse
) {
  try {
    // Forward the request to the backend with API key
    const backendUrl = `${BACKEND_URL}${req.url?.replace('/api/proxy', '') || ''}`;
    
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    // Copy safe headers from request
    for (const [key, value] of Object.entries(req.headers)) {
      if (typeof value === 'string') {
        headers[key] = value;
      }
    }

    // Add API key header (only on server side)
    if (API_KEY) {
      headers['X-API-Key'] = API_KEY;
    }

    const response = await fetch(backendUrl, {
      method: req.method,
      headers,
      body: req.method !== 'GET' ? JSON.stringify(req.body) : undefined,
    });

    const data = await response.json();
    
    res.status(response.status).json(data);
  } catch (error) {
    console.error('Proxy error:', error);
    res.status(500).json({ error: 'Proxy request failed' });
  }
}