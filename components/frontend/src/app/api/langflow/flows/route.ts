/**
 * LangFlow flows list API route
 * Proxies to backend API to get list of available flows
 */

import { NextRequest, NextResponse } from 'next/server';
import { buildForwardHeadersAsync } from '@/lib/auth';

const BACKEND_URL = process.env.BACKEND_URL || 'http://backend-service.ambient-code.svc.cluster.local:8080';

export async function GET(request: NextRequest) {
  try {
    const headers = await buildForwardHeadersAsync(request);

    const response = await fetch(`${BACKEND_URL}/api/langflow/flows`, {
      method: 'GET',
      headers,
    });

    if (!response.ok) {
      const errorText = await response.text();
      return NextResponse.json(
        { error: errorText || 'Failed to fetch flows' },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Failed to fetch LangFlow flows:', error);
    return NextResponse.json(
      { error: 'Failed to fetch flows' },
      { status: 500 }
    );
  }
}
