// Ticket types

export interface Ticket {
  id: string;
  tenant_id: string;
  title: string;
  description: string;
  priority: 'low' | 'medium' | 'high' | 'urgent';
  status: 'open' | 'in_progress' | 'waiting_customer' | 'waiting_approval' | 'resolved' | 'closed';
  category_id?: string;
  assigned_to?: string;
  assigned_by?: string;
  customer_id?: string;
  conversation_id?: string;
  channel: 'email' | 'whatsapp' | 'phone' | 'web' | 'internal';
  sla_due_at?: string;
  first_response_at?: string;
  resolution_time?: number;
  customer_satisfaction?: number;
  tags?: string[];
  metadata?: Record<string, any>;
  created_at: string;
  updated_at: string;
  resolved_at?: string;
  closed_at?: string;
}

export interface TicketCategory {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  parent_id?: string;
  color: string;
  sla_response_time: number;
  sla_resolution_time: number;
  default_priority: 'low' | 'medium' | 'high' | 'urgent';
  auto_assign_enabled: boolean;
  auto_assign_rules?: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface TicketComment {
  id: string;
  ticket_id: string;
  user_id?: string;
  customer_id?: string;
  content: string;
  comment_type: 'comment' | 'internal_note' | 'status_change' | 'assignment_change';
  visibility: 'public' | 'internal';
  attachments?: string[];
  metadata?: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface TicketAssignment {
  id: string;
  ticket_id: string;
  assigned_to: string;
  assigned_by: string;
  assigned_at: string;
  unassigned_at?: string;
  reason?: string;
  created_at: string;
  updated_at: string;
}

export interface TicketSLA {
  id: string;
  ticket_id: string;
  category_id: string;
  first_response_due: string;
  resolution_due: string;
  first_response_met: boolean;
  resolution_met: boolean;
  first_response_at?: string;
  resolved_at?: string;
  escalated_at?: string;
  escalation_level: number;
  created_at: string;
  updated_at: string;
}

export interface TicketEscalation {
  id: string;
  ticket_id: string;
  escalated_by?: string;
  escalated_to: string;
  escalation_level: number;
  reason: string;
  escalated_at: string;
  acknowledged_at?: string;
  resolved_at?: string;
  created_at: string;
  updated_at: string;
}

export interface TicketTemplate {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  title_template: string;
  content_template: string;
  category_id?: string;
  default_priority: 'low' | 'medium' | 'high' | 'urgent';
  tags?: string[];
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface TicketAuditLog {
  id: string;
  ticket_id: string;
  user_id?: string;
  action: string;
  field_name?: string;
  old_value?: string;
  new_value?: string;
  metadata?: Record<string, any>;
  created_at: string;
}

// Request/Response types
export interface CreateTicketRequest {
  title: string;
  description: string;
  priority?: 'low' | 'medium' | 'high' | 'urgent';
  category_id?: string;
  customer_id?: string;
  conversation_id?: string;
  channel?: 'email' | 'whatsapp' | 'phone' | 'web' | 'internal';
  tags?: string[];
  metadata?: Record<string, any>;
}

export interface UpdateTicketRequest {
  title?: string;
  description?: string;
  priority?: 'low' | 'medium' | 'high' | 'urgent';
  status?: 'open' | 'in_progress' | 'waiting_customer' | 'waiting_approval' | 'resolved' | 'closed';
  category_id?: string;
  assigned_to?: string;
  tags?: string[];
  metadata?: Record<string, any>;
  customer_satisfaction?: number;
}

export interface CreateTicketCommentRequest {
  content: string;
  comment_type?: 'comment' | 'internal_note';
  visibility?: 'public' | 'internal';
  attachments?: string[];
  metadata?: Record<string, any>;
}

export interface CreateTicketCategoryRequest {
  name: string;
  description?: string;
  parent_id?: string;
  color: string;
  sla_response_time: number;
  sla_resolution_time: number;
  default_priority: 'low' | 'medium' | 'high' | 'urgent';
  auto_assign_enabled: boolean;
  auto_assign_rules?: Record<string, any>;
}

export interface UpdateTicketCategoryRequest {
  name?: string;
  description?: string;
  parent_id?: string;
  color?: string;
  sla_response_time?: number;
  sla_resolution_time?: number;
  default_priority?: 'low' | 'medium' | 'high' | 'urgent';
  auto_assign_enabled?: boolean;
  auto_assign_rules?: Record<string, any>;
}

export interface TicketFilters {
  status?: string[];
  priority?: string[];
  category_id?: string;
  assigned_to?: string;
  customer_id?: string;
  channel?: string[];
  created_after?: string;
  created_before?: string;
  due_soon?: boolean;
  overdue?: boolean;
  search?: string;
}

export interface TicketStats {
  total_tickets: number;
  open_tickets: number;
  in_progress_tickets: number;
  waiting_tickets: number;
  resolved_tickets: number;
  closed_tickets: number;
  overdue_tickets: number;
  avg_resolution_time: number;
  avg_first_response_time: number;
  customer_satisfaction_avg?: number;
  tickets_by_priority: {
    low: number;
    medium: number;
    high: number;
    urgent: number;
  };
  tickets_by_category: Array<{
    category_id: string;
    category_name: string;
    count: number;
  }>;
}

// Enhanced ticket with relations
export interface TicketWithDetails extends Ticket {
  category?: TicketCategory;
  assigned_user?: {
    id: string;
    name: string;
    email: string;
  };
  customer?: {
    id: string;
    name: string;
    email?: string;
    phone?: string;
  };
  comments?: TicketComment[];
  sla?: TicketSLA;
  latest_comment?: TicketComment;
  comments_count: number;
}
