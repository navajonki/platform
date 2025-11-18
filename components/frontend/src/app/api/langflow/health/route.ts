/**
 * LangFlow health check API route
 * Proxies health check to backend API
 */

import { NextRequest, NextResponse } from 'next/server';
import { buildForwardHeadersAsync } from '@/lib/auth';

const BACKEND_URL = process.env.BACKEND_URL || 'http://backend-service.ambient-code.svc.cluster.local:8080';

export async function GET(request: NextRequest) {
  try {
    const headers = await buildForwardHeadersAsync(request);

    const response = await fetch(`${BACKEND_URL}/api/langflow/health`, {
      method: 'GET',
      headers,
    });

    if (!response.ok) {
      return NextResponse.json(
        { error: 'LangFlow service unavailable' },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('LangFlow health check failed:', error);
    return NextResponse.json(
      { error: 'Failed to check LangFlow health' },
      { status: 500 }
    );
  }
}
