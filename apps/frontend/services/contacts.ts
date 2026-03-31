import { get, post, del } from './api';

// ── Types ────────────────────────────────────────────────────────────────────

export interface TrustedContact {
  id: string;
  name: string;
  phone: string;
  relationship: string;
}

export interface TrustedContactsResponse {
  contacts: TrustedContact[];
}

export interface PendingTrustRequest {
  id: string;
  requester_id: string;
  requester_name: string;
  requester_phone: string;
  name: string;
  phone: string;
  status: 'pending' | 'accepted' | 'rejected' | 'expired';
  expires_at: string;
  created_at: string;
  accept_token?: string;
}

export interface PendingRequestsResponse {
  requests: PendingTrustRequest[];
}

export interface CreateRequestResponse {
  request: {
    id: string;
    requester_id: string;
    target_phone: string;
    name: string;
    status: string;
    expires_at: string;
    created_at: string;
  };
  accept_token: string;
}

export interface AcceptRequestResponse {
  request: PendingTrustRequest;
  contact: TrustedContact;
}

// ── Contacts API ─────────────────────────────────────────────────────────────

export async function getTrustedContacts(): Promise<TrustedContact[]> {
  try {
    const data = await get<TrustedContactsResponse>('/trusted-contacts');
    return data.contacts || [];
  } catch (err) {
    console.error('Failed to get trusted contacts:', err);
    return [];
  }
}

export async function getPendingRequests(): Promise<PendingTrustRequest[]> {
  try {
    const data = await get<PendingRequestsResponse>('/trusted-contacts/requests/pending');
    return data.requests || [];
  } catch (err) {
    console.error('Failed to get pending requests:', err);
    return [];
  }
}

export async function getOutgoingRequests(): Promise<PendingTrustRequest[]> {
  try {
    const data = await get<PendingRequestsResponse>('/trusted-contacts/requests/outgoing');
    return data.requests || [];
  } catch (err) {
    console.error('Failed to get outgoing requests:', err);
    return [];
  }
}

export async function createTrustedContactRequest(
  payload: { phone: string; email?: string }
): Promise<CreateRequestResponse> {
  return post<CreateRequestResponse>('/trusted-contacts/requests', payload);
}

export async function acceptTrustedContactRequest(
  requestId: string,
  token?: string
): Promise<AcceptRequestResponse> {
  // If authenticated, we can accept without the token (backend will verify by phone)
  return post<AcceptRequestResponse>(
    `/trusted-contacts/requests/${requestId}/accept`,
    token ? { token } : {}
  );
}

export async function rejectTrustedContactRequest(requestId: string): Promise<void> {
  await post(`/trusted-contacts/requests/${requestId}/reject`, {});
}

export async function deleteTrustedContact(contactId: string): Promise<void> {
  await del(`/trusted-contacts/${contactId}`);
}
