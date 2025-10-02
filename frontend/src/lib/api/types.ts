// Types based on Swagger API documentation

export interface PaginationParams {
  limit?: number;
  offset?: number;
  search?: string;
  archived?: boolean;
  channel_id?: string; // for alert filtering
  assigned_agent_id?: string; // for filtering by assigned agent
  has_agent?: boolean; // for filtering by conversations with/without agent
  tenant_id?: string; // for system_admin to specify tenant
  // Product filters
  category_id?: string;
  min_price?: number;
  max_price?: number;
  has_promotion?: boolean;
  has_sku?: boolean;
  has_stock?: boolean;
  out_of_stock?: boolean;
  // Order filters
  status?: string;
  payment_status?: string;
  fulfillment_status?: string;
  payment_method_id?: string;
  customer_id?: string;
  date_from?: string;
  date_to?: string;
}

export interface PaginationResult<T> {
  data: T[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

export interface User {
  id: string;
  email: string;
  name: string;
  role: string;
  phone?: string;
  is_active: boolean;
  tenant_id?: string;
  last_login_at?: string;
  created_at: string;
  updated_at: string;
}

export interface Plan {
  id: string;
  name: string;
  description: string;
  price: number;
  currency: string;
  billing_period: string;
  max_conversations: number;
  max_products: number;
  max_channels: number;
  max_messages_per_month: number;
  max_credits_per_month: number;
  features: string;
  is_active: boolean;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface Tenant {
  id: string;
  name: string;
  domain?: string;
  plan_id?: string;
  plan?: string;
  plan_name?: string; // Nome do plano da tabela plans
  plan_info?: Plan; // Objeto completo do plano
  status?: string;
  // max_users and max_messages removed - now come from plan_info
  about?: string;
  business_type?: 'sales';
  business_category?: string; // Categoria do negócio (Farmacia, Hamburgeria, Pizzaria, etc.)
  store_phone?: string; // Telefone da loja para contato
  cost_per_message?: number; // Custo em créditos por mensagem IA
  enable_ai_prompt_customization?: boolean; // Permite customização de prompts de IA
  is_public_store?: boolean; // Permite acesso público ao catálogo e informações da loja
  tag?: string; // TAG única para lojas públicas (obrigatório quando is_public_store=true)
  admin_email?: string; // Email do usuário admin do tenant
  created_at: string;
  updated_at: string;
}

export interface Channel {
  id: string;
  tenant_id: string;
  name: string;
  type: string; // whatsapp, webchat, etc.
  session: string; // session identifier for WhatsApp integration
  status?: string; // disconnected, connecting, connected
  is_active: boolean;
  config?: string;
  webhook_url?: string;
  webhook_token?: string;
  qr_code?: string;
  qr_expires_at?: string;
  conversation_count?: number; // number of conversations in this channel
  created_at: string;
  updated_at: string;
}

export interface Customer {
  id: string;
  tenant_id: string;
  name?: string;
  phone: string;
  email?: string;
  document?: string; // CPF/CNPJ
  birth_date?: string;
  gender?: string;
  notes?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Address {
  id: string;
  tenant_id: string;
  customer_id: string;
  label?: string; // home, work, etc.
  name?: string;
  street: string;
  number?: string;
  complement?: string;
  neighborhood?: string;
  city: string;
  state: string;
  zip_code: string;
  country?: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateAddressRequest {
  customer_id: string;
  label?: string;
  street: string;
  number?: string;
  complement?: string;
  neighborhood?: string;
  city: string;
  state: string;
  zip_code: string;
  country?: string;
  is_default?: boolean;
}

export interface UpdateAddressRequest {
  label?: string;
  street?: string;
  number?: string;
  complement?: string;
  neighborhood?: string;
  city?: string;
  state?: string;
  zip_code?: string;
  country?: string;
  is_default?: boolean;
}

export interface Product {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  sku?: string;
  barcode?: string;
  price: string;
  sale_price?: string;
  brand?: string;
  category_id?: string;
  weight?: string; // in grams
  dimensions?: string; // LxWxH in cm
  tags?: string;
  stock_quantity?: number;
  low_stock_threshold?: number;
  sort_order?: number;
  characteristics?: ProductCharacteristic[];
  images?: ProductMedia[];
  created_at: string;
  updated_at: string;
}

export interface ProductCharacteristic {
  id: string;
  tenant_id: string;
  product_id: string;
  title: string;
  is_required: boolean;
  is_multiple_choice: boolean;
  sort_order: number;
  items?: CharacteristicItem[];
  created_at: string;
  updated_at: string;
}

export interface CharacteristicItem {
  id: string;
  tenant_id: string;
  characteristic_id: string;
  name: string;
  price: string;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface ProductMedia {
  id: string;
  tenant_id: string;
  product_id: string;
  type: string;
  url: string;
  alt?: string;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface ProductCharacteristicCreateRequest {
  title: string;
  is_required?: boolean;
  is_multiple_choice?: boolean;
  sort_order?: number;
}

export interface ProductCharacteristicUpdateRequest {
  title?: string;
  is_required?: boolean;
  is_multiple_choice?: boolean;
  sort_order?: number;
}

export interface CharacteristicItemCreateRequest {
  name: string;
  price?: string;
  sort_order?: number;
}

export interface CharacteristicItemUpdateRequest {
  name?: string;
  price?: string;
  sort_order?: number;
}

export interface Category {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  parent_id?: string;
  image?: string;
  is_active: boolean;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface CategoryCreateRequest {
  name: string;
  description?: string;
  parent_id?: string;
  image?: string;
  sort_order?: number;
}

export interface CategoryUpdateRequest {
  name?: string;
  description?: string;
  parent_id?: string;
  image?: string;
  is_active?: boolean;
  sort_order?: number;
}

export interface ProductCreateRequest {
  name: string;
  description?: string;
  sku?: string;
  barcode?: string;
  price: string;
  sale_price?: string;
  brand?: string;
  category_id?: string;
  weight?: string;
  dimensions?: string;
  tags?: string;
  stock_quantity?: number;
  low_stock_threshold?: number;
  sort_order?: number;
}

// Product Import Types
export interface ProductImportItem {
  name: string;
  description?: string;
  price: string;
  sale_price?: string;
  sku?: string;
  barcode?: string;
  weight?: string;
  dimensions?: string;
  brand?: string;
  tags?: string;
  stock_quantity?: number;
  low_stock_threshold?: number;
  category_name?: string;
}

export interface ProductImportRequest {
  products: ProductImportItem[];
}

export interface ProductImportItemResult {
  row_number: number;
  name: string;
  sku?: string;
  status: 'created' | 'updated' | 'error';
  message: string;
  product_id?: string;
  error?: string;
}

export interface ProductImportResult {
  total_processed: number;
  created: number;
  updated: number;
  errors: number;
  results: ProductImportItemResult[];
}

// Async Import Job Types
export interface ImportJob {
  id: string;
  tenant_id: string;
  job_type: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  total_items: number;
  processed_items: number;
  successful_items: number;
  failed_items: number;
  error_message?: string;
  created_at: string;
  updated_at: string;
  completed_at?: string;
}

export interface ImportJobProgress {
  job: ImportJob;
  progress_percentage: number;
  details?: ProductImportResult;
}

export interface CreateImportJobResponse {
  job_id: string;
  message: string;
}

export interface ProductImageImportResult {
  success: boolean;
  created_count?: number;
  credits_used?: number;
  products?: {
    name: string;
    description?: string;
    price?: string;
    tags?: string;
    status?: string;
  }[];
  errors?: string[];
}

export interface Order {
  id: string;
  tenant_id: string;
  customer_id?: string;
  payment_method_id?: string;
  order_number?: string;
  status?: string;
  payment_status?: string;
  fulfillment_status?: string;
  total_amount?: string;
  subtotal?: string;
  tax_amount?: string;
  shipping_amount?: string;
  discount_amount?: string;
  currency?: string;
  notes?: string;
  shipped_at?: string;
  delivered_at?: string;
  created_at: string;
  updated_at: string;
  
  // Historical customer data
  customer_name?: string;
  customer_email?: string;
  customer_phone?: string;
  customer_document?: string;
  
  // Historical shipping address data
  shipping_name?: string;
  shipping_street?: string;
  shipping_number?: string;
  shipping_complement?: string;
  shipping_neighborhood?: string;
  shipping_city?: string;
  shipping_state?: string;
  shipping_zipcode?: string;
  shipping_country?: string;
  
  // Payment observations and change
  observations?: string;
  change_for?: string;
  
  // Historical billing address data
  billing_name?: string;
  billing_street?: string;
  billing_number?: string;
  billing_complement?: string;
  billing_neighborhood?: string;
  billing_city?: string;
  billing_state?: string;
  billing_zipcode?: string;
  billing_country?: string;
  
  customer?: Customer;
  payment_method?: PaymentMethod;
  items?: OrderItem[];
}

export interface OrderItem {
  id?: string;
  tenant_id?: string;
  order_id?: string;
  product_id: string;
  quantity: number;
  price: string;
  total: string;
  
  // Historical product data for order integrity
  product_name?: string;
  product_description?: string;
  product_sku?: string;
  product_category_id?: string;
  product_category_name?: string;
  unit_price?: string;
  
  // Item attributes
  attributes?: OrderItemAttribute[];
  
  created_at?: string;
  updated_at?: string;
}

export interface OrderItemAttribute {
  id: string;
  order_item_id: string;
  attribute_id: string;
  option_id: string;
  attribute_name: string;
  option_name: string;
  option_price: string;
}

export interface OrderWithCustomer extends Order {
  customer_name?: string;
  customer_email?: string;
  payment_method_name?: string;
  items_count?: number;
}

export interface PaymentMethod {
  id: string;
  name: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface OrderStats {
  total_orders: number;
  pending_orders: number;
  delivered_today: number;
  revenue: number;
  growth_rates?: {
    orders: number;
    revenue: number;
  };
  status_breakdown?: Record<string, number>;
  recent_orders?: Array<{
    id: string;
    order_number: string;
    customer_name: string;
    status: string;
    payment_status: string;
    total_amount: number;
    items_count: number;
    created_at: string;
  }>;
}

export interface ProductStats {
  total: number;
  with_sku: number;
  with_promotion: number;
  average_price: number;
  total_value: number;
  categories_count: number;
  out_of_stock: number;
  recent_added: number; // last 7 days
}

// AI Credits interfaces
export interface AICredits {
  id: string;
  tenant_id: string;
  total_credits: number;
  used_credits: number;
  remaining_credits: number;
  last_updated_by?: string;
  created_at: string;
  updated_at: string;
}

export interface AICreditTransaction {
  id: string;
  tenant_id: string;
  user_id: string;
  type: 'add' | 'use' | 'refund';
  amount: number;
  description: string;
  related_entity?: string;
  related_id?: string;
  created_at: string;
  updated_at: string;
}

export interface ProductGenerationResponse {
  description: string;
  tags: string[];
  suggestions: {
    brand?: string;
    category?: string;
    keywords?: string[];
  };
}

export interface ProductGenerationResult {
  product_info: ProductGenerationResponse;
  credits_used: number;
  remaining_credits: number;
  message: string;
}

export interface ProductGenerationRequest {
  product_name: string;
}

export interface AddCreditsRequest {
  tenant_id: string;
  amount: number;
  description: string;
}

export interface UseCreditsRequest {
  amount: number;
  description: string;
  related_entity?: string;
  related_id?: string;
}

export interface Message {
  id: string;
  tenant_id: string;
  conversation_id?: string;
  customer_id?: string;
  user_id?: string; // null for incoming messages
  user_name?: string; // name of the user who sent the message
  type?: string; // text, image, audio, video, document
  content?: string;
  direction?: string; // in, out, note
  status?: string; // sent, delivered, read, failed
  external_id?: string;
  webhook_id?: string;
  reply_to_id?: string;
  forwarded_from?: string;
  metadata?: string;
  media_url?: string;
  media_type?: string;
  filename?: string;
  is_note?: boolean; // true for internal notes
  created_at: string;
  updated_at: string;
}

export interface Customer {
  id: string;
  tenant_id: string;
  phone: string;
  name?: string;
  email?: string;
  avatar?: string;
  metadata?: string;
  created_at: string;
  updated_at: string;
}

export interface Conversation {
  id: string;
  tenant_id: string;
  customer_id: string;
  channel_id: string;
  status: string; // open, closed
  priority: string; // low, normal, high, urgent
  assigned_agent_id?: string;
  unread_count: number;
  last_message_at?: string;
  is_archived: boolean;
  is_pinned: boolean;
  ai_enabled: boolean;
  created_at: string;
  updated_at: string;
  // Relations
  customer?: Customer;
  channel?: Channel;
  last_message?: Message;
  assigned_agent?: User; // Added assigned agent relation
}

export interface ConversationWithMessages {
  conversation: Conversation;
  messages: Message[];
}

export interface ConversationCounts {
  novas: number;
  em_atendimento: number;
  minhas: number;
  arquivadas: number;
}

export interface ReportData {
  period: string;
  revenue: number;
  orders: number;
  customers: number;
  products_sold: number;
}

export interface ComparisonMetadata {
  has_sufficient_data: boolean;
  comparison_period: string;
  data_points: number;
  message: string;
}

export interface ReportsResponse {
  data: {
    monthly?: ReportData[];
    daily?: ReportData[];
  } | ReportData[];
  comparison: ComparisonMetadata;
}

export interface TopProductReport {
  product_id: string;
  product_name: string;
  sales_count: number;
  total_revenue: number;
}

export interface SalesAnalytics {
  total_revenue: number;
  total_orders: number;
  new_customers: number;
  average_ticket: number;
  monthly_data: ReportData[];
  top_products: TopProductReport[];
  conversion_rate: number;
  growth_rate: number;
}

// Auth types
export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: User;
}

export interface SendMessageRequest {
  conversation_id: string;
  type: 'text' | 'image' | 'audio' | 'video' | 'document';
  content: string;
  reply_to_id?: string;
  resend_message_id?: string;
}

export interface CreateNoteRequest {
  conversation_id: string;
  content: string;
}

// Message Template types
export interface MessageTemplate {
  id: string;
  tenant_id: string;
  user_id: string;
  title: string;
  content: string;
  variables: string; // JSON string of variable names
  category?: string;
  is_active: boolean;
  usage_count: number;
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateTemplateRequest {
  title: string;
  content: string;
  variables?: string[];
  category?: string;
  description?: string;
}

export interface UpdateTemplateRequest {
  title: string;
  content: string;
  variables?: string[];
  category?: string;
  description?: string;
}

export interface ProcessTemplateRequest {
  template_id: string;
  variables: Record<string, string>;
}

export interface ProcessTemplateResponse {
  processed_content: string;
  original_content: string;
  template_title: string;
}

// WhatsApp types
export interface WhatsAppSessionStatus {
  name: string;
  status: 'SCAN_QR_CODE' | 'WORKING' | 'STOPPED' | 'FAILED';
  config?: any;
  me?: {
    id: string;
    pushName: string;
  };
  engine?: {
    engine: string;
    WWebVersion: string;
    state: string;
  };
}

// API Response wrapper
export interface ApiResponse<T> {
  data: T;
  success: boolean;
  message?: string;
}

// Error class
export class ApiError extends Error {
  public status: number;
  public data?: any;

  constructor({ message, status, data }: { message: string; status: number; data?: any }) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.data = data;
  }
}

// Alert types
export interface Alert {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  channel_id: string;
  group_name: string;
  group_id?: string;
  phones?: string; // comma-separated phone numbers
  trigger_on: string; // 'order_created', 'message_received', etc.
  is_active: boolean;
  created_at: string;
  updated_at: string;
  channel?: Channel; // populated relation
}

export interface CreateAlertRequest {
  name: string;
  description?: string;
  channel_id: string;
  group_name: string;
  phones?: string;
  trigger_on?: string;
  is_active?: boolean;
}

// Notification types
export interface NotificationLog {
  id: string;
  tenant_id: string;
  tenant_name: string;
  type: string;
  recipient: string;
  subject: string;
  body: string;
  status: 'sent' | 'failed' | 'pending';
  sent_at: string;
  error_message?: string;
}

export interface NotificationStats {
  total: number;
  sent: number;
  failed: number;
  pending: number;
  by_tenant: Array<{ tenant_id: string; tenant_name: string; count: number }>;
  by_status: Array<{ status: string; count: number }>;
  by_type: Array<{ type: string; count: number }>;
}

export interface NotificationResendRequest {
  notification_id: string;
  tenant_ids: string[];
  force_resend?: boolean;
}

export interface NotificationTriggerRequest {
  type: string;
  tenant_ids: string[];
  force_run: boolean;
}

// Dashboard statistics types
export interface DashboardStats {
  total_products: number;
  products_on_sale: number;
  total_customers: number;
  active_customers: number;
  total_orders: number;
  pending_orders: number;
  total_channels: number;
  active_channels: number;
  connected_channels: number;
}

