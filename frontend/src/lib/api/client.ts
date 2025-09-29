import { 
  ApiError, 
  PaginationParams, 
  PaginationResult, 
  LoginRequest, 
  LoginResponse, 
  User,
  Tenant,
  Customer,
  Address,
  CreateAddressRequest,
  UpdateAddressRequest,
  Product,
  ProductCreateRequest,
  ProductImportRequest,
  ProductImportResult,
  ProductImageImportResult,
  ProductCharacteristic,
  ProductCharacteristicCreateRequest,
  ProductCharacteristicUpdateRequest,
  CharacteristicItem,
  CharacteristicItemCreateRequest,
  CharacteristicItemUpdateRequest,
  ProductMedia,
  Category,
  CategoryCreateRequest,
  CategoryUpdateRequest,
  ImportJob,
  ImportJobProgress,
  CreateImportJobResponse,
  Order,
  OrderWithCustomer,
  OrderStats,
  PaymentMethod,
  ProductStats,
  Channel,
  Alert,
  CreateAlertRequest,
  Message,
  SendMessageRequest,
  CreateNoteRequest,
  MessageTemplate,
  CreateTemplateRequest,
  UpdateTemplateRequest,
  ProcessTemplateRequest,
  ProcessTemplateResponse,
  Conversation,
  ConversationWithMessages,
  ConversationCounts,
  ReportData,
  ReportsResponse,
  ComparisonMetadata,
  TopProductReport,
  SalesAnalytics,
  WhatsAppSessionStatus,
  DashboardStats,
  AICredits,
  AICreditTransaction,
  ProductGenerationResult,
  ProductGenerationRequest,
  AddCreditsRequest,
  UseCreditsRequest,
  NotificationLog,
  NotificationStats,
  NotificationResendRequest,
  NotificationTriggerRequest,
  
} from './types';

export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

class ApiClient {
  private token: string | null = null;

  constructor() {
    // Load token from localStorage on initialization
    this.token = localStorage.getItem('access_token');
  }

  setToken(token: string) {
    this.token = token;
    localStorage.setItem('access_token', token);
  }

  clearToken() {
    this.token = null;
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${API_BASE_URL}${endpoint}`;
    
    // Don't set Content-Type for FormData, let browser handle it
    const isFormData = options.body instanceof FormData;
    
    const config: RequestInit = {
      headers: {
        ...(isFormData ? {} : { 'Content-Type': 'application/json' }),
        ...(this.token && { Authorization: `Bearer ${this.token}` }),
        ...options.headers,
      },
      ...options,
    };

    try {
      const response = await fetch(url, config);
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new ApiError({
          message: errorData.error || errorData.message || `HTTP error! status: ${response.status}`,
          status: response.status,
          data: errorData, // Include the complete error response data
        });
      }

      // Check if response has content before trying to parse JSON
      const contentLength = response.headers.get('content-length');
      const contentType = response.headers.get('content-type');
      
      if (contentLength === '0' || !contentType?.includes('application/json')) {
        // Return empty object for successful responses without content
        return {} as T;
      }

      const data = await response.json();
      
      // Check if the response contains an error field even with 200 status
      // Only treat as error if 'error' is a string (error message), not a number (data field)
      if (data && typeof data === 'object' && 'error' in data && typeof data.error === 'string') {
        throw new ApiError({
          message: data.error,
          status: response.status,
          data: data, // Include the full response data for detailed error handling
        });
      }

      return data;
    } catch (error) {
      if (error instanceof ApiError) {
        throw error;
      }
      throw new ApiError({
        message: error instanceof Error ? error.message : 'Network error',
        status: 0,
      });
    }
  }

  // Auth endpoints
  async login(credentials: LoginRequest): Promise<LoginResponse> {
    const response = await this.request<LoginResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(credentials),
    });
    
    this.setToken(response.access_token);
    localStorage.setItem('refresh_token', response.refresh_token);
    
    return response;
  }

  async refreshToken(): Promise<LoginResponse> {
    const refreshToken = localStorage.getItem('refresh_token');
    if (!refreshToken) {
      throw new Error('No refresh token available');
    }

    const response = await this.request<LoginResponse>('/auth/refresh', {
      method: 'POST',
      body: JSON.stringify({ refresh_token: refreshToken }),
    });
    
    this.setToken(response.access_token);
    localStorage.setItem('refresh_token', response.refresh_token);
    
    return response;
  }

  logout() {
    this.clearToken();
  }

  async forgotPassword(email: string): Promise<{ message: string }> {
    return this.request<{ message: string }>('/auth/forgot-password', {
      method: 'POST',
      body: JSON.stringify({ email }),
    });
  }

  async resetPassword(token: string, newPassword: string): Promise<{ message: string }> {
    return this.request<{ message: string }>('/auth/reset-password', {
      method: 'POST',
      body: JSON.stringify({ token, new_password: newPassword }),
    });
  }

  // User profile endpoints
  async updateProfile(profile: { name: string; email: string; phone?: string }): Promise<User> {
    return this.request<User>('/auth/profile', {
      method: 'PUT',
      body: JSON.stringify(profile),
    });
  }

  async changePassword(data: { current_password: string; new_password: string }): Promise<{ message: string }> {
    return this.request<{ message: string }>('/auth/change-password', {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  // Dashboard endpoints
  async getUnreadMessagesCount(): Promise<{ unread_count: number }> {
    return this.request<{ unread_count: number }>('/dashboard/unread-messages');
  }

  async getDashboardStats(): Promise<DashboardStats> {
    return this.request<DashboardStats>('/dashboard/stats');
  }


  async getProductStats(): Promise<ProductStats> {
    return this.request<ProductStats>('/products/stats');
  }

  async getConversationCounts(): Promise<ConversationCounts> {
    return this.request<ConversationCounts>('/dashboard/conversation-counts');
  }

  // Tenant endpoints (admin only)
  async getTenants(params: PaginationParams = {}): Promise<PaginationResult<Tenant>> {
    const queryParams = new URLSearchParams();
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.offset) queryParams.append('offset', params.offset.toString());
    
    const endpoint = `/admin/tenants${queryParams.toString() ? `?${queryParams}` : ''}`;
    return this.request<PaginationResult<Tenant>>(endpoint);
  }

  async getTenantsStats(): Promise<{
    total_tenants: number;
    active_tenants: number;
    paid_plan_tenants: number;
    total_conversations: number;
  }> {
    return this.request<{
      total_tenants: number;
      active_tenants: number;
      paid_plan_tenants: number;
      total_conversations: number;
    }>('/admin/tenants/stats');
  }

  async createTenant(tenant: Omit<Tenant, 'id' | 'created_at' | 'updated_at'>): Promise<Tenant> {
    return this.request<Tenant>('/admin/tenants', {
      method: 'POST',
      body: JSON.stringify(tenant),
    });
  }

  async getTenant(id: string): Promise<Tenant> {
    return this.request<Tenant>(`/admin/tenants/${id}`);
  }

  async updateTenant(id: string, tenant: Partial<Tenant>): Promise<Tenant> {
    console.log(`🏢 API: updateTenant(${id})`, tenant);
    return this.request<Tenant>(`/admin/tenants/${id}`, {
      method: 'PUT',
      body: JSON.stringify(tenant),
    });
  }

  async deleteTenant(id: string): Promise<{ message: string }> {
    return this.request<{ message: string }>(`/admin/tenants/${id}`, {
      method: 'DELETE',
    });
  }

  async getTenantStats(id: string): Promise<{
    total_conversations: number;
    active_conversations: number;
    total_messages: number;
    messages_current_month: number;
  }> {
    return this.request<{
      total_conversations: number;
      active_conversations: number;
      total_messages: number;
      messages_current_month: number;
    }>(`/admin/tenants/${id}/stats`);
  }

  // Export tenant products (super_admin only)
  async getTenantsForExport(): Promise<Array<{
    id: string;
    name: string;
    created_at: string;
    product_count: number;
  }>> {
    return this.request<Array<{
      id: string;
      name: string;
      created_at: string;
      product_count: number;
    }>>('/admin/export/tenants');
  }

  async exportTenantProducts(tenantId: string): Promise<Blob> {
    const url = `${API_BASE_URL}/admin/export/tenants/${tenantId}/products`;
    
    const response = await fetch(url, {
      method: 'GET',
      headers: {
        ...(this.token && { Authorization: `Bearer ${this.token}` }),
      },
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({}));
      throw new Error(error.message || `HTTP error! status: ${response.status}`);
    }

    return response.blob();
  }

  // Create tenant admin (system_admin only)
  async createTenantAdmin(data: {
    tenant_id: string;
    name: string;
    email: string;
    phone?: string;
    password: string;
  }): Promise<{ message: string; user: User }> {
    return this.request<{ message: string; user: User }>('/admin/tenant-admin', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  // Get users of a specific tenant (system_admin only)
  async getTenantUsers(tenantId: string, params: PaginationParams = {}): Promise<{
    users: User[];
    pagination: {
      page: number;
      limit: number;
      total: number;
      pages: number;
    };
  }> {
    const queryParams = new URLSearchParams();
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.offset) queryParams.append('offset', params.offset.toString());
    
    const endpoint = `/users${queryParams.toString() ? `?${queryParams}` : ''}`;
    return this.request(endpoint);
  }

  // Get tenant users for system admin
  async getTenantUsersForAdmin(tenantId: string, params: PaginationParams = {}): Promise<{
    users: User[];
    pagination: {
      page: number;
      limit: number;
      total: number;
      pages: number;
    };
  }> {
    const queryParams = new URLSearchParams();
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.offset) queryParams.append('offset', params.offset.toString());
    
    const endpoint = `/admin/tenants/${tenantId}/users${queryParams.toString() ? `?${queryParams}` : ''}`;
    return this.request(endpoint);
  }

  // Update tenant admin (system_admin only)
  async updateTenantAdmin(userId: string, data: {
    name: string;
    email: string;
    phone?: string;
    password?: string;
  }): Promise<{ message: string; user: User }> {
    console.log(`👤 API: updateTenantAdmin(${userId})`, data);
    return this.request<{ message: string; user: User }>(`/admin/tenant-admin/${userId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  // System admin statistics
  async getSystemStats(): Promise<any> {
    return this.request<any>('/admin/stats');
  }

  // Channel endpoints
  async getChannels(params: PaginationParams = {}): Promise<PaginationResult<Channel>> {
    const queryParams = new URLSearchParams();
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.offset) queryParams.append('offset', params.offset.toString());
    
    const endpoint = `/channels${queryParams.toString() ? `?${queryParams}` : ''}`;
    return this.request<PaginationResult<Channel>>(endpoint);
  }

  async createChannel(channel: Omit<Channel, 'id' | 'tenant_id' | 'created_at' | 'updated_at'>): Promise<Channel> {
    return this.request<Channel>('/channels', {
      method: 'POST',
      body: JSON.stringify(channel),
    });
  }

  async getChannel(id: string): Promise<Channel> {
    return this.request<Channel>(`/channels/${id}`);
  }

  async updateChannel(id: string, channel: Partial<Channel>): Promise<Channel> {
    return this.request<Channel>(`/channels/${id}`, {
      method: 'PUT',
      body: JSON.stringify(channel),
    });
  }

  async deleteChannel(id: string): Promise<void> {
    await this.request(`/channels/${id}`, {
      method: 'DELETE',
    });
  }

  async migrateConversations(channelId: string, destinationChannelId: string): Promise<{ message: string }> {
    return this.request<{ message: string }>(`/channels/${channelId}/migrate-conversations`, {
      method: 'POST',
      body: JSON.stringify({ destination_channel_id: destinationChannelId }),
    });
  }

  // Admin Channel endpoints (System Admin only)
  async getChannelsByTenant(tenantId: string, params: PaginationParams = {}): Promise<PaginationResult<Channel>> {
    const queryParams = new URLSearchParams();
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.offset) queryParams.append('offset', params.offset.toString());
    
    const endpoint = `/admin/tenants/${tenantId}/channels${queryParams.toString() ? `?${queryParams}` : ''}`;
    return this.request<PaginationResult<Channel>>(endpoint);
  }

  async createChannelForTenant(tenantId: string, channel: Omit<Channel, 'id' | 'tenant_id' | 'created_at' | 'updated_at'>): Promise<Channel> {
    return this.request<Channel>(`/admin/tenants/${tenantId}/channels`, {
      method: 'POST',
      body: JSON.stringify(channel),
    });
  }

  async updateChannelForTenant(tenantId: string, channelId: string, channel: Partial<Channel>): Promise<Channel> {
    return this.request<Channel>(`/admin/tenants/${tenantId}/channels/${channelId}`, {
      method: 'PUT',
      body: JSON.stringify(channel),
    });
  }

  async deleteChannelForTenant(tenantId: string, channelId: string): Promise<{ message: string }> {
    return this.request<{ message: string }>(`/admin/tenants/${tenantId}/channels/${channelId}`, {
      method: 'DELETE',
    });
  }

  // Customer endpoints
  async getCustomers(params: PaginationParams = {}): Promise<PaginationResult<Customer>> {
    const queryParams = new URLSearchParams();
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.offset) queryParams.append('offset', params.offset.toString());
    if (params.search) queryParams.append('search', params.search);
    
    const endpoint = `/customers${queryParams.toString() ? `?${queryParams}` : ''}`;
    return this.request<PaginationResult<Customer>>(endpoint);
  }

  async createCustomer(customer: Omit<Customer, 'id' | 'tenant_id' | 'created_at' | 'updated_at'>): Promise<Customer> {
    return this.request<Customer>('/customers', {
      method: 'POST',
      body: JSON.stringify(customer),
    });
  }

  async getCustomer(id: string): Promise<Customer> {
    return this.request<Customer>(`/customers/${id}`);
  }

  async updateCustomer(id: string, customer: Partial<Customer>): Promise<Customer> {
    return this.request<Customer>(`/customers/${id}`, {
      method: 'PUT',
      body: JSON.stringify(customer),
    });
  }

  // Address endpoints
  async getAddressesByCustomer(customerId: string): Promise<{ addresses: Address[]; total: number }> {
    return this.request<{ addresses: Address[]; total: number }>(`/addresses/customer/${customerId}`);
  }

  async createAddress(address: CreateAddressRequest): Promise<Address> {
    return this.request<Address>('/addresses', {
      method: 'POST',
      body: JSON.stringify(address),
    });
  }

  async getAddress(id: string): Promise<Address> {
    return this.request<Address>(`/addresses/${id}`);
  }

  async updateAddress(id: string, address: UpdateAddressRequest): Promise<Address> {
    return this.request<Address>(`/addresses/${id}`, {
      method: 'PUT',
      body: JSON.stringify(address),
    });
  }

  async deleteAddress(id: string): Promise<{ message: string }> {
    return this.request<{ message: string }>(`/addresses/${id}`, {
      method: 'DELETE',
    });
  }

  async setDefaultAddress(id: string): Promise<{ message: string }> {
    return this.request<{ message: string }>(`/addresses/${id}/set-default`, {
      method: 'POST',
    });
  }

  // Product endpoints
  async getProducts(params: PaginationParams = {}): Promise<PaginationResult<Product>> {
    const queryParams = new URLSearchParams();
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.offset) queryParams.append('offset', params.offset.toString());
    if (params.search) queryParams.append('search', params.search);
    if (params.archived !== undefined) queryParams.append('archived', params.archived.toString());
    
    const endpoint = `/products${queryParams.toString() ? `?${queryParams}` : ''}`;
    return this.request<PaginationResult<Product>>(endpoint);
  }

  async getProductsAdmin(params: PaginationParams = {}): Promise<PaginationResult<Product>> {
    const queryParams = new URLSearchParams();
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.offset) queryParams.append('offset', params.offset.toString());
    if (params.search) queryParams.append('search', params.search);
    if (params.archived !== undefined) queryParams.append('archived', params.archived.toString());
    
    // Product filters
    if (params.category_id) queryParams.append('category_id', params.category_id);
    if (params.min_price !== undefined) queryParams.append('min_price', params.min_price.toString());
    if (params.max_price !== undefined) queryParams.append('max_price', params.max_price.toString());
    if (params.has_promotion !== undefined) queryParams.append('has_promotion', params.has_promotion.toString());
    if (params.has_sku !== undefined) queryParams.append('has_sku', params.has_sku.toString());
    if (params.has_stock !== undefined) queryParams.append('has_stock', params.has_stock.toString());
    if (params.out_of_stock !== undefined) queryParams.append('out_of_stock', params.out_of_stock.toString());
    
    const endpoint = `/products/admin${queryParams.toString() ? `?${queryParams}` : ''}`;
    return this.request<PaginationResult<Product>>(endpoint);
  }

  async createProduct(product: ProductCreateRequest): Promise<Product> {
    return this.request<Product>('/products', {
      method: 'POST',
      body: JSON.stringify(product),
    });
  }

  async getProduct(id: string): Promise<Product> {
    return this.request<Product>(`/products/${id}`);
  }

  async updateProduct(id: string, product: Partial<Product>): Promise<Product> {
    return this.request<Product>(`/products/${id}`, {
      method: 'PUT',
      body: JSON.stringify(product),
    });
  }

  async deleteProduct(id: string): Promise<void> {
    return this.request<void>(`/products/${id}`, {
      method: 'DELETE',
    });
  }

  // Product Characteristics endpoints
  async getProductCharacteristics(productId: string): Promise<ProductCharacteristic[]> {
    return this.request<ProductCharacteristic[]>(`/products/${productId}/characteristics`);
  }

  async getProductCharacteristic(productId: string, id: string): Promise<ProductCharacteristic> {
    return this.request<ProductCharacteristic>(`/products/${productId}/characteristics/${id}`);
  }

  async createProductCharacteristic(productId: string, characteristic: ProductCharacteristicCreateRequest): Promise<ProductCharacteristic> {
    return this.request<ProductCharacteristic>(`/products/${productId}/characteristics`, {
      method: 'POST',
      body: JSON.stringify(characteristic),
    });
  }

  async updateProductCharacteristic(productId: string, id: string, characteristic: ProductCharacteristicUpdateRequest): Promise<ProductCharacteristic> {
    return this.request<ProductCharacteristic>(`/products/${productId}/characteristics/${id}`, {
      method: 'PUT',
      body: JSON.stringify(characteristic),
    });
  }

  async deleteProductCharacteristic(productId: string, id: string): Promise<void> {
    return this.request<void>(`/products/${productId}/characteristics/${id}`, {
      method: 'DELETE',
    });
  }

  // Characteristic Items endpoints
  async getCharacteristicItems(characteristicId: string): Promise<CharacteristicItem[]> {
    return this.request<CharacteristicItem[]>(`/characteristics/${characteristicId}/items`);
  }

  async createCharacteristicItem(characteristicId: string, item: CharacteristicItemCreateRequest): Promise<CharacteristicItem> {
    return this.request<CharacteristicItem>(`/characteristics/${characteristicId}/items`, {
      method: 'POST',
      body: JSON.stringify(item),
    });
  }

  async updateCharacteristicItem(characteristicId: string, id: string, item: CharacteristicItemUpdateRequest): Promise<CharacteristicItem> {
    return this.request<CharacteristicItem>(`/characteristics/${characteristicId}/items/${id}`, {
      method: 'PUT',
      body: JSON.stringify(item),
    });
  }

  async deleteCharacteristicItem(characteristicId: string, id: string): Promise<void> {
    return this.request<void>(`/characteristics/${characteristicId}/items/${id}`, {
      method: 'DELETE',
    });
  }

  // Product Images endpoints
  async getProductImages(productId: string): Promise<ProductMedia[]> {
    return this.request<ProductMedia[]>(`/products/${productId}/images`);
  }

  async uploadProductImage(productId: string, file: File): Promise<{ id: string; url: string; message: string }> {
    const formData = new FormData();
    formData.append('image', file);
    
    return this.request<{ id: string; url: string; message: string }>(`/products/${productId}/upload-image`, {
      method: 'POST',
      body: formData,
    });
  }

  async deleteProductImage(productId: string, imageId: string): Promise<void> {
    return this.request<void>(`/products/${productId}/images/${imageId}`, {
      method: 'DELETE',
    });
  }

  async importProducts(importRequest: ProductImportRequest): Promise<ProductImportResult> {
    return this.request<ProductImportResult>('/products/import', {
      method: 'POST',
      body: JSON.stringify(importRequest),
    });
  }

  async importProductsFromCSV(formData: FormData): Promise<ProductImportResult> {
    return this.request<ProductImportResult>('/products/import', {
      method: 'POST',
      body: formData,
    });
  }

  async importProductsFromImage(formData: FormData): Promise<ProductImageImportResult> {
    return this.request<ProductImageImportResult>('/products/import-image', {
      method: 'POST',
      body: formData,
    });
  }

  // Category endpoints
  async getCategories(): Promise<Category[]> {
    return this.request<Category[]>('/categories');
  }

  async createCategory(category: CategoryCreateRequest): Promise<Category> {
    return this.request<Category>('/categories', {
      method: 'POST',
      body: JSON.stringify(category),
    });
  }

  async getCategory(id: string): Promise<Category> {
    return this.request<Category>(`/categories/${id}`);
  }

  async updateCategory(id: string, category: CategoryUpdateRequest): Promise<Category> {
    return this.request<Category>(`/categories/${id}`, {
      method: 'PUT',
      body: JSON.stringify(category),
    });
  }

  async deleteCategory(id: string): Promise<void> {
    return this.request<void>(`/categories/${id}`, {
      method: 'DELETE',
    });
  }

  async getRootCategories(): Promise<Category[]> {
    return this.request<Category[]>('/categories/root');
  }

  async getCategoriesByParent(parentId: string): Promise<Category[]> {
    return this.request<Category[]>(`/categories/parent/${parentId}`);
  }

  // Async Import Jobs endpoints
  async createProductImportJob(formData: FormData): Promise<CreateImportJobResponse> {
    return this.request<CreateImportJobResponse>('/import/products', {
      method: 'POST',
      body: formData,
    });
  }

  async getImportJobProgress(jobId: string): Promise<ImportJobProgress> {
    return this.request<ImportJobProgress>(`/import/jobs/${jobId}/progress`);
  }

  // AI Credits endpoints
  async getAICredits(): Promise<AICredits> {
    return this.request<AICredits>('/ai-credits');
  }

  async useAICredits(request: UseCreditsRequest): Promise<{ message: string; credits: AICredits }> {
    return this.request<{ message: string; credits: AICredits }>('/ai-credits/use', {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  async getAICreditTransactions(page = 1, limit = 20): Promise<{ transactions: AICreditTransaction[]; page: number; limit: number }> {
    return this.request<{ transactions: AICreditTransaction[]; page: number; limit: number }>(`/ai-credits/transactions?page=${page}&limit=${limit}`);
  }

  async addAICredits(request: AddCreditsRequest): Promise<{ message: string; credits: AICredits }> {
    return this.request<{ message: string; credits: AICredits }>('/admin/ai-credits/add', {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  // Admin methods for managing tenant credits
  async getAICreditsForTenant(tenantId: string): Promise<AICredits> {
    return this.request<AICredits>(`/admin/ai-credits/tenant/${tenantId}`);
  }

  async getAICreditTransactionsForTenant(tenantId: string, page = 1, limit = 20): Promise<{ transactions: AICreditTransaction[]; page: number; limit: number }> {
    return this.request<{ transactions: AICreditTransaction[]; page: number; limit: number }>(`/admin/ai-credits/tenant/${tenantId}/transactions?page=${page}&limit=${limit}`);
  }

  async addAICreditsToTenant(request: AddCreditsRequest): Promise<{ message: string; credits: AICredits }> {
    return this.request<{ message: string; credits: AICredits }>('/admin/ai-credits/add', {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  // Admin Notification Management endpoints
  async getNotifications(params: PaginationParams & {
    tenant_id?: string;
    type?: string;
    status?: string;
    start_date?: string;
    end_date?: string;
  } = {}): Promise<PaginationResult<NotificationLog>> {
    const queryParams = new URLSearchParams();
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.offset) queryParams.append('offset', params.offset.toString());
    if (params.tenant_id) queryParams.append('tenant_id', params.tenant_id);
    if (params.type) queryParams.append('type', params.type);
    if (params.status) queryParams.append('status', params.status);
    if (params.start_date) queryParams.append('start_date', params.start_date);
    if (params.end_date) queryParams.append('end_date', params.end_date);
    
    const endpoint = `/admin/notifications${queryParams.toString() ? `?${queryParams}` : ''}`;
    return this.request<PaginationResult<NotificationLog>>(endpoint);
  }

  async getNotificationStats(params: {
    tenant_id?: string;
    type?: string;
    status?: string;
    start_date?: string;
    end_date?: string;
  } = {}): Promise<NotificationStats> {
    const queryParams = new URLSearchParams();
    if (params.tenant_id) queryParams.append('tenant_id', params.tenant_id);
    if (params.type) queryParams.append('type', params.type);
    if (params.status) queryParams.append('status', params.status);
    if (params.start_date) queryParams.append('start_date', params.start_date);
    if (params.end_date) queryParams.append('end_date', params.end_date);
    
    const endpoint = `/admin/notifications/stats${queryParams.toString() ? `?${queryParams}` : ''}`;
    return this.request<NotificationStats>(endpoint);
  }

  async resendNotifications(request: NotificationResendRequest): Promise<{ message: string }> {
    return this.request<{ message: string }>('/admin/notifications/resend', {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  async triggerNotifications(request: NotificationTriggerRequest): Promise<{ message: string }> {
    return this.request<{ message: string }>('/admin/notifications/trigger', {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  // AI Product Generation endpoints
  async generateProductInfo(request: ProductGenerationRequest): Promise<ProductGenerationResult> {
    return this.request<ProductGenerationResult>('/ai/products/generate', {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  async getProductGenerationEstimate(request: ProductGenerationRequest): Promise<{
    estimated_credits: number;
    product_name: string;
  }> {
    return this.request<{
      estimated_credits: number;
      product_name: string;
    }>('/ai/products/estimate', {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  async downloadImportTemplate(): Promise<string> {
    const response = await fetch(`${API_BASE_URL}/products/import/template`, {
      headers: {
        ...(this.token && { Authorization: `Bearer ${this.token}` }),
      },
    });
    
    if (!response.ok) {
      throw new ApiError({
        message: `Failed to download template: ${response.status}`,
        status: response.status,
      });
    }
    
    return await response.text();
  }

  // Order endpoints
  async getOrders(params: PaginationParams = {}): Promise<PaginationResult<OrderWithCustomer>> {
    const queryParams = new URLSearchParams();
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.offset) queryParams.append('offset', params.offset.toString());
    if (params.search) queryParams.append('search', params.search);
    if (params.status) queryParams.append('status', params.status);
    if (params.payment_status) queryParams.append('payment_status', params.payment_status);
    if (params.fulfillment_status) queryParams.append('fulfillment_status', params.fulfillment_status);
    if (params.payment_method_id) queryParams.append('payment_method_id', params.payment_method_id);
    if (params.customer_id) queryParams.append('customer_id', params.customer_id);
    if (params.date_from) queryParams.append('date_from', params.date_from);
    if (params.date_to) queryParams.append('date_to', params.date_to);
    
    const endpoint = `/orders${queryParams.toString() ? `?${queryParams}` : ''}`;
    return this.request<PaginationResult<OrderWithCustomer>>(endpoint);
  }

  async createOrder(order: Omit<Order, 'id' | 'created_at' | 'updated_at'>): Promise<Order> {
    return this.request<Order>('/orders', {
      method: 'POST',
      body: JSON.stringify(order),
    });
  }

  async getOrder(id: string): Promise<Order> {
    return this.request<Order>(`/orders/${id}`);
  }

  async updateOrder(id: string, order: Partial<Order>): Promise<Order> {
    return this.request<Order>(`/orders/${id}`, {
      method: 'PUT',
      body: JSON.stringify(order),
    });
  }

  async sendOrderEmail(data: { order_id: string; recipient: string; subject: string; message: string }): Promise<{ success: boolean; message: string }> {
    return this.request<{ success: boolean; message: string }>('/orders/send-email', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  // Order Item endpoints
  async addOrderItem(orderId: string, item: { 
    product_id: string; 
    quantity: number; 
    price?: string;
    attributes?: Array<{
      attribute_id: string;
      option_id: string;
      attribute_name: string;
      option_name: string;
      option_price: string;
    }>
  }): Promise<Order> {
    return this.request<Order>(`/orders/${orderId}/items`, {
      method: 'POST',
      body: JSON.stringify(item),
    });
  }

  async updateOrderItem(orderId: string, itemId: string, item: { quantity?: number; price?: string }): Promise<Order> {
    return this.request<Order>(`/orders/${orderId}/items/${itemId}`, {
      method: 'PUT',
      body: JSON.stringify(item),
    });
  }

  async removeOrderItem(orderId: string, itemId: string): Promise<Order> {
    return this.request<Order>(`/orders/${orderId}/items/${itemId}`, {
      method: 'DELETE',
    });
  }

  // Message endpoints
  async getMessages(conversationId: string, params: PaginationParams = {}): Promise<Message[]> {
    const queryParams = new URLSearchParams();
    queryParams.append('conversation_id', conversationId);
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.offset) queryParams.append('offset', params.offset.toString());
    
    const endpoint = `/messages?${queryParams}`;
    return this.request<Message[]>(endpoint);
  }

  async createMessage(message: Omit<Message, 'id' | 'created_at' | 'updated_at'>): Promise<Message> {
    return this.request<Message>('/messages', {
      method: 'POST',
      body: JSON.stringify(message),
    });
  }

  async getMessage(id: string): Promise<Message> {
    return this.request<Message>(`/messages/${id}`);
  }

  // WhatsApp endpoints
  async sendWhatsAppMessage(request: SendMessageRequest): Promise<Message> {
    return this.request<Message>('/whatsapp/send', {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  async updateMessageStatus(messageId: string, status: string): Promise<Message> {
    return this.request<Message>(`/whatsapp/messages/${messageId}/status`, {
      method: 'PUT',
      body: JSON.stringify({ status }),
    });
  }

  async createNote(request: CreateNoteRequest): Promise<Message> {
    return this.request<Message>('/messages/notes', {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  // Message Templates endpoints
  async getMessageTemplates(params: PaginationParams & { category?: string; search?: string } = {}): Promise<MessageTemplate[]> {
    const queryParams = new URLSearchParams();
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.offset) queryParams.append('offset', params.offset.toString());
    if (params.category) queryParams.append('category', params.category);
    if (params.search) queryParams.append('search', params.search);
    
    const endpoint = `/message-templates${queryParams.toString() ? `?${queryParams}` : ''}`;
    return this.request<MessageTemplate[]>(endpoint);
  }

  async getMessageTemplate(id: string): Promise<MessageTemplate> {
    return this.request<MessageTemplate>(`/message-templates/${id}`);
  }

  async createMessageTemplate(request: CreateTemplateRequest): Promise<MessageTemplate> {
    return this.request<MessageTemplate>('/message-templates', {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  async updateMessageTemplate(id: string, request: UpdateTemplateRequest): Promise<MessageTemplate> {
    return this.request<MessageTemplate>(`/message-templates/${id}`, {
      method: 'PUT',
      body: JSON.stringify(request),
    });
  }

  async deleteMessageTemplate(id: string): Promise<void> {
    await this.request(`/message-templates/${id}`, {
      method: 'DELETE',
    });
  }

  async getTemplateCategories(): Promise<string[]> {
    return this.request<string[]>('/message-templates/categories');
  }

  async processTemplate(request: ProcessTemplateRequest): Promise<ProcessTemplateResponse> {
    return this.request<ProcessTemplateResponse>('/message-templates/process', {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  async getConversations(params: PaginationParams = {}): Promise<{ conversations: Conversation[], pagination: any }> {
    const queryParams = new URLSearchParams();
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.offset) queryParams.append('offset', params.offset.toString());
    if (params.search) queryParams.append('search', params.search);
    if (params.archived !== undefined) queryParams.append('archived', params.archived.toString());
    if (params.assigned_agent_id) queryParams.append('assigned_agent_id', params.assigned_agent_id);
    if (params.has_agent !== undefined) queryParams.append('has_agent', params.has_agent.toString());
    
    const endpoint = `/whatsapp/conversations${queryParams.toString() ? `?${queryParams}` : ''}`;
    return this.request<{ conversations: Conversation[], pagination: any }>(endpoint);
  }

  async getConversation(id: string): Promise<ConversationWithMessages> {
    return this.request<ConversationWithMessages>(`/whatsapp/conversations/${id}?messages_limit=50`);
  }

  async markConversationAsRead(id: string): Promise<void> {
    await this.request(`/whatsapp/conversations/${id}/read`, {
      method: 'POST',
    });
  }

  async archiveConversation(id: string): Promise<void> {
    await this.request(`/whatsapp/conversations/${id}/archive`, {
      method: 'POST',
    });
  }

  async pinConversation(id: string): Promise<void> {
    await this.request(`/whatsapp/conversations/${id}/pin`, {
      method: 'POST',
    });
  }

  async toggleAIConversation(id: string): Promise<{ ai_enabled: boolean; success: boolean }> {
    const response = await this.request(`/whatsapp/conversations/${id}/toggle-ai`, {
      method: 'POST',
    });
    return response as { ai_enabled: boolean; success: boolean };
  }

  async updateConversationStatus(id: string, updates: {
    status?: string;
    priority?: string;
    is_archived?: boolean;
    is_pinned?: boolean;
    assigned_agent_id?: string | null;
  }): Promise<void> {
    await this.request(`/whatsapp/conversations/${id}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    });
  }

  async assignConversation(id: string, agentId: string | null): Promise<void> {
    await this.request(`/whatsapp/conversations/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ assigned_agent_id: agentId }),
    });
  }

  async findOrCreateConversationByCustomer(customerId: string): Promise<Conversation> {
    return this.request<Conversation>(`/conversations/customer/${customerId}`, {
      method: 'POST',
    });
  }

  // Analytics/Reports endpoints
  async getSalesAnalytics(params: { 
    start_date?: string; 
    end_date?: string; 
    period?: 'daily' | 'weekly' | 'monthly' 
  } = {}): Promise<SalesAnalytics> {
    const queryParams = new URLSearchParams();
    if (params.start_date) queryParams.append('start_date', params.start_date);
    if (params.end_date) queryParams.append('end_date', params.end_date);
    if (params.period) queryParams.append('period', params.period);
    
    const endpoint = `/analytics/sales${queryParams.toString() ? `?${queryParams}` : ''}`;
    return this.request<SalesAnalytics>(endpoint);
  }

  async getOrderStats(): Promise<OrderStats> {
    return this.request<OrderStats>('/analytics/orders');
  }

  async getReportsData(params: { 
    type: 'revenue' | 'orders' | 'customers' | 'products';
    period?: 'daily' | 'weekly' | 'monthly';
    start_date?: string;
    end_date?: string;
  }): Promise<ReportsResponse> {
    const queryParams = new URLSearchParams();
    queryParams.append('type', params.type);
    if (params.period) queryParams.append('period', params.period);
    if (params.start_date) queryParams.append('start_date', params.start_date);
    if (params.end_date) queryParams.append('end_date', params.end_date);
    
    const endpoint = `/reports${queryParams.toString() ? `?${queryParams}` : ''}`;
    return this.request<ReportsResponse>(endpoint);
  }

  async getTopProducts(params: { 
    limit?: number;
    period?: 'daily' | 'weekly' | 'monthly';
    start_date?: string;
    end_date?: string;
  } = {}): Promise<TopProductReport[]> {
    const queryParams = new URLSearchParams();
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.period) queryParams.append('period', params.period);
    if (params.start_date) queryParams.append('start_date', params.start_date);
    if (params.end_date) queryParams.append('end_date', params.end_date);
    
    const endpoint = `/reports/top-products${queryParams.toString() ? `?${queryParams}` : ''}`;
    return this.request<TopProductReport[]>(endpoint);
  }

  // WhatsApp Integration (via backend proxy)
  async getWhatsAppSessionStatus(channelId: string): Promise<WhatsAppSessionStatus> {
    const response = await this.request<WhatsAppSessionStatus>(`/whatsapp/session-status?channel_id=${channelId}`);
    return response;
  }

  async getWhatsAppQR(channelId: string, tenantId?: string): Promise<string> {
    let url = `${API_BASE_URL}/whatsapp/qr?channel_id=${channelId}`;
    
    // Add tenant_id parameter for system_admin
    if (tenantId) {
      url += `&tenant_id=${tenantId}`;
    }
    
    const response = await fetch(url, {
      method: 'GET',
      headers: {
        'accept': 'image/png',
        ...(this.token && { Authorization: `Bearer ${this.token}` }),
      },
    });
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `Failed to get QR code: ${response.status}`);
    }
    
    const blob = await response.blob();
    return URL.createObjectURL(blob);
  }

  async disconnectWhatsApp(channelId: string): Promise<{ message: string }> {
    return this.request<{ message: string }>(`/whatsapp/disconnect?channel_id=${channelId}`, {
      method: 'POST',
    });
  }

  // Error Logs endpoints
  async getErrorLogs(params: {
    page?: number;
    limit?: number;
    severity?: string;
    error_type?: string;
    resolved?: string;
  } = {}): Promise<{
    error_logs: Array<{
      id: string;
      tenant_id: string;
      customer_phone: string;
      customer_id?: string;
      user_message: string;
      tool_name: string;
      tool_args: string;
      error_message: string;
      error_type: string;
      user_response: string;
      severity: 'error' | 'warning' | 'critical';
      resolved: boolean;
      resolved_at?: string;
      resolved_by?: string;
      stack_trace?: string;
      created_at: string;
      updated_at: string;
    }>;
    pagination: {
      page: number;
      limit: number;
      total: number;
      pages: number;
    };
  }> {
    const queryParams = new URLSearchParams();
    
    if (params.page) queryParams.append('page', params.page.toString());
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.severity) queryParams.append('severity', params.severity);
    if (params.error_type) queryParams.append('error_type', params.error_type);
    if (params.resolved) queryParams.append('resolved', params.resolved);

    const endpoint = `/admin/error-logs${queryParams.toString() ? `?${queryParams}` : ''}`;
    return this.request(endpoint);
  }

  async getErrorStats(): Promise<{
    total_errors: number;
    unresolved_errors: number;
    critical_errors: number;
    errors_by_type: Array<{
      type: string;
      count: number;
    }>;
  }> {
    return this.request('/admin/error-logs/stats');
  }

  async resolveError(errorId: string): Promise<{ message: string }> {
    return this.request(`/admin/error-logs/${errorId}/resolve`, {
      method: 'PUT',
    });
  }

  // Tenant Settings endpoints
  async getTenantSettings(): Promise<{ settings: any[] }> {
    return this.request('/settings');
  }

  async getTenantSetting(key: string): Promise<{ setting: any }> {
    return this.request(`/settings/${key}`);
  }

  async updateTenantSetting(key: string, value: string, settingType: string = 'text'): Promise<{ success: boolean }> {
    return this.request(`/settings/${key}`, {
      method: 'PUT',
      body: JSON.stringify({ value, setting_type: settingType }),
    });
  }

  async generateAIPrompt(): Promise<{ success: boolean; prompt: string; examples: string[]; welcomeMessage: string; message: string }> {
    return this.request('/settings/ai/generate-auto-prompt', {
      method: 'POST',
    });
  }

  async generateAIExamples(): Promise<{ success: boolean; examples: string[]; message: string }> {
    return this.request('/settings/ai/generate-examples', {
      method: 'POST',
    });
  }

  async generateWelcomeMessage(): Promise<{ success: boolean; message: string }> {
    return this.request('/settings/ai/generate-welcome', {
      method: 'POST',
    });
  }

  async resetAIToDefault(): Promise<{ success: boolean; message: string }> {
    return this.request('/settings/ai/reset-to-default', {
      method: 'POST',
    });
  }

  // Sales AI - Context Limitation
  async getSalesContextLimitation(): Promise<{ contextLimitation: string }> {
    try {
      const response: any = await this.request('/settings/ai_context_limitation_custom');
      return { contextLimitation: response.setting?.setting_value || '' };
    } catch (error) {
      // Se não existe setting, retornar padrão
      return { contextLimitation: '' };
    }
  }

  async setSalesContextLimitation(contextLimitation: string): Promise<{ success: boolean; message: string }> {
    await this.updateTenantSetting('ai_context_limitation_custom', contextLimitation, 'text');
    return { success: true, message: 'Configuração salva com sucesso!' };
  }

  async resetSalesContextLimitation(): Promise<{ success: boolean; message: string }> {
    const defaultText = `🚨 LIMITAÇÃO DE CONTEXTO - SUPER IMPORTANTE:
- Você é um ASSISTENTE DE VENDAS, não um assistente geral
- NUNCA responda perguntas sobre: política, notícias, medicina, direito, aposentadoria, educação, tecnologia geral, ou qualquer assunto não relacionado à nossa loja
- Para perguntas fora do contexto, responda: "Sou um assistente focado em vendas da nossa loja. Como posso ajudá-lo com nossos produtos ou serviços?"
- SEMPRE redirecione conversas para produtos, pedidos, entregas ou informações da loja
- Sua função é EXCLUSIVAMENTE ajudar com vendas e atendimento comercial`;
    
    await this.updateTenantSetting('ai_context_limitation_custom', defaultText, 'text');
    return { success: true, message: 'Configuração restaurada ao padrão!' };
  }


  // Tenant Profile Management
  async getTenantProfile(): Promise<Tenant> {
    return this.request('/tenant/profile');
  }


  async updateTenantProfile(data: Partial<Tenant>): Promise<Tenant> {
    return this.request('/tenant/profile', {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  // User management endpoints
  async getUsers(params: { page?: number; limit?: number } = {}): Promise<{
    users: User[];
    pagination: {
      page: number;
      limit: number;
      total: number;
      pages: number;
    };
  }> {
    const queryParams = new URLSearchParams();
    if (params.page) queryParams.append('page', params.page.toString());
    if (params.limit) queryParams.append('limit', params.limit.toString());
    
    const endpoint = `/users${queryParams.toString() ? `?${queryParams}` : ''}`;
    return this.request(endpoint);
  }

  async getUser(id: string): Promise<{ user: User }> {
    return this.request(`/users/${id}`);
  }

  async createUser(userData: {
    name: string;
    email: string;
    phone: string;
    password: string;
  }): Promise<{ message: string; user: User }> {
    return this.request('/users', {
      method: 'POST',
      body: JSON.stringify(userData),
    });
  }

  async updateUser(id: string, userData: {
    name: string;
    email: string;
    phone: string;
    is_active: boolean;
  }): Promise<{ message: string; user: User }> {
    return this.request(`/users/${id}`, {
      method: 'PUT',
      body: JSON.stringify(userData),
    });
  }

  async changeUserPassword(id: string, password: string): Promise<{ message: string }> {
    return this.request(`/users/${id}/password`, {
      method: 'PUT',
      body: JSON.stringify({ password }),
    });
  }

  async deleteUser(id: string): Promise<{ message: string }> {
    return this.request(`/users/${id}`, {
      method: 'DELETE',
    });
  }

  // Conversation users management endpoints
  async getConversationUsers(conversationId: string): Promise<{ users: User[] }> {
    return this.request(`/conversations/${conversationId}/users`);
  }

  async assignUserToConversation(conversationId: string, userId: string, session: string): Promise<{ success: boolean; message: string }> {
    return this.request(`/conversations/${conversationId}/assign-user`, {
      method: 'POST',
      body: JSON.stringify({
        user_id: userId,
        session: session,
      }),
    });
  }

  async unassignUserFromConversation(conversationId: string, userId: string, session: string): Promise<{ success: boolean; message: string }> {
    return this.request(`/conversations/${conversationId}/unassign-user/${userId}?session=${session}`, {
      method: 'DELETE',
    });
  }

  async getWhatsAppGroupProxyStatus(): Promise<{ enabled: boolean }> {
    return this.request('/tenant/whatsapp-group-proxy-status');
  }

  // Alert endpoints
  async getAlerts(params: PaginationParams = {}): Promise<PaginationResult<Alert>> {
    const searchParams = new URLSearchParams();
    if (params.limit) searchParams.append('limit', params.limit.toString());
    if (params.offset) searchParams.append('page', Math.floor(params.offset / (params.limit || 20) + 1).toString());
    if (params.search) searchParams.append('search', params.search);
    if (params.channel_id) searchParams.append('channel_id', params.channel_id);

    const url = `/alerts${searchParams.toString() ? '?' + searchParams.toString() : ''}`;
    const response = await this.request<{ alerts: Alert[], pagination: any }>(url);
    
    return {
      data: response.alerts,
      total: response.pagination.total,
      page: response.pagination.page,
      per_page: response.pagination.limit,
      total_pages: response.pagination.totalPages,
    };
  }

  async createAlert(alert: CreateAlertRequest): Promise<Alert> {
    return this.request<Alert>('/alerts', {
      method: 'POST',
      body: JSON.stringify(alert),
    });
  }

  async getAlert(id: string): Promise<Alert> {
    return this.request<Alert>(`/alerts/${id}`);
  }

  async updateAlert(id: string, alert: Partial<CreateAlertRequest>): Promise<Alert> {
    return this.request<Alert>(`/alerts/${id}`, {
      method: 'PUT',
      body: JSON.stringify(alert),
    });
  }

  async deleteAlert(id: string): Promise<void> {
    await this.request(`/alerts/${id}`, {
      method: 'DELETE',
    });
  }

  // Delivery APIs
  async getDeliveryStoreLocation(): Promise<any> {
    return this.request('/delivery/store-location');
  }

  async updateDeliveryStoreLocation(data: {
    store_street: string;
    store_number?: string;
    store_neighborhood?: string;
    store_city: string;
    store_state: string;
    store_zip_code?: string;
    store_country?: string;
    delivery_radius_km: number;
  }): Promise<any> {
    return this.request('/delivery/store-location', {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async getDeliveryZones(): Promise<{ zones: any[] }> {
    return this.request('/delivery/zones');
  }

  async manageDeliveryZone(data: {
    neighborhood: string;
    city: string;
    state: string;
    zone_type: 'whitelist' | 'blacklist';
    action: 'add' | 'remove';
  }): Promise<any> {
    return this.request('/delivery/zones', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async validateDeliveryAddress(data: {
    street: string;
    number?: string;
    neighborhood?: string;
    city: string;
    state: string;
    zip_code?: string;
    country?: string;
  }): Promise<any> {
    return this.request('/delivery/validate', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  // Media upload endpoints
  async uploadMedia(formData: FormData, type?: string): Promise<{ url: string; messageId: string }> {
    const url = type ? `/whatsapp/upload/media?type=${encodeURIComponent(type)}` : '/whatsapp/upload/media';
    return this.request<{ url: string; messageId: string }>(url, {
      method: 'POST',
      body: formData,
    });
  }

  // WhatsApp media sending endpoints
  async sendWhatsAppImage(data: {
    chatId: string;
    file: {
      mimetype: string;
      filename: string;
      url: string;
    };
    caption?: string;
    conversation_id: string;
  }): Promise<{ _data: { id: string } }> {
    return this.request<{ _data: { id: string } }>('/whatsapp/send/image', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async sendWhatsAppDocument(data: {
    chatId: string;
    file: {
      mimetype: string;
      filename: string;
      url: string;
    };
    caption?: string;
    conversation_id: string;
  }): Promise<{ _data: { id: string } }> {
    return this.request<{ _data: { id: string } }>('/whatsapp/send/document', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async sendWhatsAppAudio(data: {
    chatId: string;
    file: {
      mimetype: string;
      url: string;
    };
    convert?: boolean;
    conversation_id: string;
  }): Promise<{ _data: { id: string } }> {
    return this.request<{ _data: { id: string } }>('/whatsapp/send/audio', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  // Plans management endpoints
  async getPlans(): Promise<any[]> {
    return this.request<any[]>('/admin/plans');
  }

  async getPlan(id: string): Promise<any> {
    return this.request<any>(`/admin/plans/${id}`);
  }

  async createPlan(plan: any): Promise<any> {
    return this.request<any>('/admin/plans', {
      method: 'POST',
      body: JSON.stringify(plan),
    });
  }

  async updatePlan(id: string, plan: any): Promise<any> {
    return this.request<any>(`/admin/plans/${id}`, {
      method: 'PUT',
      body: JSON.stringify(plan),
    });
  }

  async deletePlan(id: string): Promise<void> {
    return this.request<void>(`/admin/plans/${id}`, {
      method: 'DELETE',
    });
  }

  async updateTenantPlan(tenantId: string, planId: string): Promise<{ message: string }> {
    return this.request<{ message: string }>(`/admin/tenants/${tenantId}/plan/${planId}`, {
      method: 'PUT',
    });
  }

  async getTenantUsage(tenantId?: string): Promise<any> {
    const endpoint = tenantId ? `/admin/tenants/${tenantId}/usage` : '/billing/usage';
    return this.request<any>(endpoint);
  }

  async checkUsageLimit(resourceType: string, amount?: number): Promise<any> {
    const params = new URLSearchParams({
      resource_type: resourceType,
      ...(amount && { amount: amount.toString() }),
    });
    return this.request<any>(`/billing/usage/check?${params}`);
  }

  async incrementUsage(resourceType: string, amount: number): Promise<{ message: string }> {
    return this.request<{ message: string }>(`/billing/usage/${resourceType}/increment`, {
      method: 'POST',
      body: JSON.stringify({ amount }),
    });
  }


  // Payment Methods
  async getPaymentMethods(params: {
    page?: number;
    limit?: number;
    search?: string;
    show_inactive?: boolean;
  } = {}): Promise<{
    data: PaymentMethod[];
    total: number;
    page: number;
    limit: number;
  }> {
    const queryParams = new URLSearchParams();
    if (params.page) queryParams.append('page', params.page.toString());
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.search) queryParams.append('search', params.search);
    if (params.show_inactive) queryParams.append('show_inactive', 'true');
    
    const endpoint = `/payment-methods${queryParams.toString() ? `?${queryParams}` : ''}`;
    return this.request(endpoint);
  }

  async getActivePaymentMethods(): Promise<PaymentMethod[]> {
    const response = await this.request<{success: boolean, data: PaymentMethod[]}>('/payment-methods/active');
    return response.data || [];
  }

  async createPaymentMethod(data: {
    name: string;
    is_active?: boolean;
  }): Promise<PaymentMethod> {
    return this.request('/payment-methods', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async getPaymentMethod(id: string): Promise<PaymentMethod> {
    return this.request(`/payment-methods/${id}`);
  }

  async updatePaymentMethod(id: string, data: {
    name?: string;
    is_active?: boolean;
  }): Promise<PaymentMethod> {
    return this.request(`/payment-methods/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deletePaymentMethod(id: string): Promise<void> {
    return this.request(`/payment-methods/${id}`, {
      method: 'DELETE',
    });
  }




  // Onboarding endpoints
  async getOnboardingStatus(): Promise<{
    is_completed: boolean;
    completion_rate: number;
    items: Array<{
      id: string;
      title: string;
      description: string;
      is_completed: boolean;
      action_url: string;
      priority: number;
      completed_at?: string;
    }>;
    tenant_created_at: string;
  }> {
    return this.request('/onboarding/status');
  }

  async dismissOnboarding(): Promise<{ message: string }> {
    return this.request('/onboarding/dismiss', {
      method: 'POST',
    });
  }

  async completeOnboardingItem(itemId: string): Promise<{ message: string }> {
    return this.request(`/onboarding/complete/${itemId}`, {
      method: 'POST',
    });
  }

  // Municipios methods
  async getEstados(search?: string): Promise<Array<{ uf: string; nome: string }>> {
    const params = new URLSearchParams();
    if (search) {
      params.append('search', search);
    }
    const url = `/municipios/estados${params.toString() ? `?${params.toString()}` : ''}`;
    return this.request(url);
  }

  async getCidades(uf: string, search?: string): Promise<Array<{
    id: number;
    nome_cidade: string;
    uf: string;
    ddd: number;
    latitude: number;
    longitude: number;
  }>> {
    const params = new URLSearchParams();
    params.append('uf', uf);
    if (search) {
      params.append('search', search);
    }
    const url = `/municipios/cidades?${params.toString()}`;
    return this.request(url);
  }
}

// Create a singleton instance
export const apiClient = new ApiClient();

// Export the class for potential multiple instances
export { ApiClient };
