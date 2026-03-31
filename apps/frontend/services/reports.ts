import { get, post } from './api';

export interface ComplaintEvent {
    id: string;
    status: string;
    actor: string;
    note: string | null;
    created_at: string;
}

export interface ReportHistory {
    id: string;
    user_id: string;
    type: string;
    description: string | null;
    latitude: number;
    longitude: number;
    occurred_at: string;
    created_at: string;
    status: string;
    events: ComplaintEvent[];
    evidence_ids?: string[];
}

export interface CreateReportInput {
    type: string;
    description: string;
    lat: number;
    lng: number;
}

export async function getUserReports(): Promise<ReportHistory[]> {
    const response = await get<{ reports: ReportHistory[] }>('/reports/me');
    return response.reports;
}

export async function createReport(input: CreateReportInput): Promise<any> {
    return await post('/reports', input);
}
