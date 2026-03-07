import { NextResponse } from 'next/server';

// GET /api/health - Health check endpoint for monitoring
export async function GET() {
  const healthStatus = {
    status: 'healthy',
    timestamp: new Date().toISOString(),
    service: 'claoj-frontend',
    version: process.env.NEXT_PUBLIC_VERSION || 'unknown',
    uptime: process.uptime(),
  };

  return NextResponse.json(healthStatus);
}

// HEAD /api/health - Lightweight health check
export async function HEAD() {
  return new NextResponse(null, {
    status: 200,
    headers: {
      'Content-Type': 'application/json',
    },
  });
}
