import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from './client';
import { useAuth } from '@/contexts/AuthContext';
import { 
  Customer, 
  Address,
  CreateAddressRequest,
  UpdateAddressRequest,
  Product, 
  ProductCreateRequest,
  ProductImportRequest,
  ProductImportResult,
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
  Order,
  OrderWithCustomer,
  OrderStats,
  Channel, 
  Message, 
  Tenant,
  PaginationParams,
  PaginationResult,
  SendMessageRequest,
  CreateNoteRequest,
  MessageTemplate,
  CreateTemplateRequest,
  UpdateTemplateRequest,
  ProcessTemplateRequest,
  Alert,
  CreateAlertRequest,
  Conversation,
  ConversationWithMessages,
  SalesAnalytics,
  ReportData,
  TopProductReport,
  DashboardStats,
  ProductStats,
  ConversationCounts
} from './types';

// Query keys
export const queryKeys = {
  customers: ['customers'] as const,
  customer: (id: string) => ['customers', id] as const,
  addresses: (customerId: string) => ['addresses', 'customer', customerId] as const,
  address: (id: string) => ['addresses', id] as const,
  products: ['products'] as const,
  product: (id: string) => ['products', id] as const,
  productCharacteristics: (productId: string) => ['products', productId, 'characteristics'] as const,
  productCharacteristic: (productId: string, id: string) => ['products', productId, 'characteristics', id] as const,
  characteristicItems: (characteristicId: string) => ['characteristics', characteristicId, 'items'] as const,
  productImages: (productId: string) => ['products', productId, 'images'] as const,
  categories: ['categories'] as const,
  category: (id: string) => ['categories', id] as const,
  orders: ['orders'] as const,
  order: (id: string) => ['orders', id] as const,
  channels: ['channels'] as const,
  channel: (id: string) => ['channels', id] as const,
  messages: (conversationId: string) => ['messages', conversationId] as const,
  messageTemplates: ['message-templates'] as const,
  messageTemplate: (id: string) => ['message-templates', id] as const,
  templateCategories: ['template-categories'] as const,
  tenants: ['tenants'] as const,
  tenant: (id: string) => ['tenants', id] as const,
  conversations: ['conversations'] as const,
  conversation: (id: string) => ['conversations', id] as const,
  analytics: ['analytics'] as const,
  reports: (type: string) => ['reports', type] as const,
  topProducts: ['top-products'] as const,
  unreadMessages: ['unread-messages'] as const,
  dashboardStats: ['dashboard-stats'] as const,
  conversationCounts: ['conversation-counts'] as const,
  alerts: ['alerts'] as const,
  alert: (id: string) => ['alerts', id] as const,
};

// Customer hooks
export const useCustomers = (params: PaginationParams = {}) => {
  const { user } = useAuth();
  
  return useQuery({
    queryKey: [...queryKeys.customers, params],
    queryFn: () => apiClient.getCustomers(params),
    staleTime: 1000 * 60 * 5, // 5 minutos - maior cache para evitar re-fetches
    refetchOnWindowFocus: false, // Evitar refetch ao focar janela
    enabled: user?.role !== 'system_admin', // Skip for system_admin
  });
};

export const useCustomer = (id: string) => {
  return useQuery({
    queryKey: queryKeys.customer(id),
    queryFn: () => apiClient.getCustomer(id),
    enabled: !!id,
  });
};

export const useCreateCustomer = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (customer: Omit<Customer, 'id' | 'tenant_id' | 'created_at' | 'updated_at'>) =>
      apiClient.createCustomer(customer),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.customers });
    },
  });
};

export const useUpdateCustomer = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, customer }: { id: string; customer: Partial<Customer> }) =>
      apiClient.updateCustomer(id, customer),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.customers });
      queryClient.invalidateQueries({ queryKey: queryKeys.customer(id) });
    },
  });
};

// Address hooks
export const useAddressesByCustomer = (customerId: string, options?: { enabled?: boolean }) => {
  return useQuery({
    queryKey: queryKeys.addresses(customerId),
    queryFn: () => apiClient.getAddressesByCustomer(customerId),
    staleTime: 1000 * 60, // 1 minute
    enabled: options?.enabled ?? !!customerId,
  });
};

export const useAddress = (id: string) => {
  return useQuery({
    queryKey: queryKeys.address(id),
    queryFn: () => apiClient.getAddress(id),
    enabled: !!id,
  });
};

export const useCreateAddress = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (address: CreateAddressRequest) =>
      apiClient.createAddress(address),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.addresses(data.customer_id) });
    },
  });
};

export const useUpdateAddress = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, address }: { id: string; address: UpdateAddressRequest }) =>
      apiClient.updateAddress(id, address),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.addresses(data.customer_id) });
      queryClient.invalidateQueries({ queryKey: queryKeys.address(data.id) });
    },
  });
};

export const useDeleteAddress = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, customerId }: { id: string; customerId: string }) =>
      apiClient.deleteAddress(id),
    onSuccess: (_, { customerId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.addresses(customerId) });
    },
  });
};

export const useSetDefaultAddress = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, customerId }: { id: string; customerId: string }) =>
      apiClient.setDefaultAddress(id),
    onSuccess: (_, { customerId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.addresses(customerId) });
    },
  });
};

// Product hooks
export const useProducts = (params: PaginationParams = {}) => {
  const { user } = useAuth();
  
  return useQuery({
    queryKey: [...queryKeys.products, params],
    queryFn: () => apiClient.getProducts(params),
    staleTime: 1000 * 60 * 2, // 2 minutes - cache data to avoid unnecessary requests
    gcTime: 1000 * 60 * 5, // 5 minutes - keep in cache
    enabled: !!user, // Only when user is available - tenant_id comes from JWT for tenant_admin
  });
};

export const useProductsAdmin = (params: PaginationParams = {}) => {
  const { user } = useAuth();
  
  return useQuery({
    queryKey: [...queryKeys.products, 'admin', params],
    queryFn: () => apiClient.getProductsAdmin(params),
    staleTime: 1000 * 60 * 2, // 2 minutes
    gcTime: 1000 * 60 * 5, // 5 minutes
    enabled: !!user, // Only when user is available - tenant_id comes from JWT for tenant_admin
  });
};

export const useProduct = (id: string) => {
  return useQuery({
    queryKey: queryKeys.product(id),
    queryFn: () => apiClient.getProduct(id),
    enabled: !!id && id !== "skip",
  });
};

export const useCreateProduct = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (product: ProductCreateRequest) =>
      apiClient.createProduct(product),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.products });
    },
  });
};

export const useUpdateProduct = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, product }: { id: string; product: Partial<Product> }) =>
      apiClient.updateProduct(id, product),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.products });
      queryClient.invalidateQueries({ queryKey: queryKeys.product(id) });
    },
  });
};

export const useDeleteProduct = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: string) => apiClient.deleteProduct(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.products });
    },
  });
};

// Product Characteristics hooks
export const useProductCharacteristics = (productId: string) => {
  return useQuery({
    queryKey: queryKeys.productCharacteristics(productId),
    queryFn: () => apiClient.getProductCharacteristics(productId),
    enabled: !!productId,
  });
};

export const useProductCharacteristic = (productId: string, id: string) => {
  return useQuery({
    queryKey: queryKeys.productCharacteristic(productId, id),
    queryFn: () => apiClient.getProductCharacteristic(productId, id),
    enabled: !!productId && !!id,
  });
};

export const useCreateProductCharacteristic = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ productId, characteristic }: { productId: string; characteristic: ProductCharacteristicCreateRequest }) =>
      apiClient.createProductCharacteristic(productId, characteristic),
    onSuccess: (_, { productId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.productCharacteristics(productId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.product(productId) });
    },
  });
};

export const useUpdateProductCharacteristic = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ productId, id, characteristic }: { productId: string; id: string; characteristic: ProductCharacteristicUpdateRequest }) =>
      apiClient.updateProductCharacteristic(productId, id, characteristic),
    onSuccess: (_, { productId, id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.productCharacteristics(productId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.productCharacteristic(productId, id) });
      queryClient.invalidateQueries({ queryKey: queryKeys.product(productId) });
    },
  });
};

export const useDeleteProductCharacteristic = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ productId, id }: { productId: string; id: string }) =>
      apiClient.deleteProductCharacteristic(productId, id),
    onSuccess: (_, { productId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.productCharacteristics(productId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.product(productId) });
    },
  });
};

// Characteristic Items hooks
export const useCharacteristicItems = (characteristicId: string) => {
  return useQuery({
    queryKey: queryKeys.characteristicItems(characteristicId),
    queryFn: () => apiClient.getCharacteristicItems(characteristicId),
    enabled: !!characteristicId,
  });
};

export const useCreateCharacteristicItem = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ characteristicId, item }: { characteristicId: string; item: CharacteristicItemCreateRequest }) =>
      apiClient.createCharacteristicItem(characteristicId, item),
    onSuccess: (_, { characteristicId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.characteristicItems(characteristicId) });
      // Invalidate product queries that include this characteristic
      queryClient.invalidateQueries({ queryKey: queryKeys.products });
    },
  });
};

export const useUpdateCharacteristicItem = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ characteristicId, id, item }: { characteristicId: string; id: string; item: CharacteristicItemUpdateRequest }) =>
      apiClient.updateCharacteristicItem(characteristicId, id, item),
    onSuccess: (_, { characteristicId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.characteristicItems(characteristicId) });
      // Invalidate product queries that include this characteristic
      queryClient.invalidateQueries({ queryKey: queryKeys.products });
    },
  });
};

export const useDeleteCharacteristicItem = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ characteristicId, id }: { characteristicId: string; id: string }) =>
      apiClient.deleteCharacteristicItem(characteristicId, id),
    onSuccess: (_, { characteristicId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.characteristicItems(characteristicId) });
      // Invalidate product queries that include this characteristic
      queryClient.invalidateQueries({ queryKey: queryKeys.products });
    },
  });
};

// Product Images hooks
export const useProductImages = (productId: string) => {
  return useQuery({
    queryKey: queryKeys.productImages(productId),
    queryFn: () => apiClient.getProductImages(productId),
    enabled: !!productId,
  });
};

export const useUploadProductImage = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ productId, file }: { productId: string; file: File }) =>
      apiClient.uploadProductImage(productId, file),
    onSuccess: (_, { productId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.productImages(productId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.product(productId) });
    },
  });
};

export const useDeleteProductImage = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ productId, imageId }: { productId: string; imageId: string }) =>
      apiClient.deleteProductImage(productId, imageId),
    onSuccess: (_, { productId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.productImages(productId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.product(productId) });
    },
  });
};

// Category hooks
export const useCategories = () => {
  return useQuery({
    queryKey: queryKeys.categories,
    queryFn: () => apiClient.getCategories(),
  });
};

export const useCategory = (id: string) => {
  return useQuery({
    queryKey: queryKeys.category(id),
    queryFn: () => apiClient.getCategory(id),
    enabled: !!id,
  });
};

export const useCreateCategory = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (category: CategoryCreateRequest) => apiClient.createCategory(category),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.categories });
    },
  });
};

export const useUpdateCategory = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: CategoryUpdateRequest }) => apiClient.updateCategory(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.categories });
    },
  });
};

export const useDeleteCategory = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: string) => apiClient.deleteCategory(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.categories });
    },
  });
};

export const useImportProducts = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (importRequest: ProductImportRequest) => 
      apiClient.importProducts(importRequest),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.products });
    },
  });
};

export const useImportProductsFromCSV = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (formData: FormData) => 
      apiClient.importProductsFromCSV(formData),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.products });
    },
  });
};

export const useImportProductsFromImage = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (formData: FormData) => 
      apiClient.importProductsFromImage(formData),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.products });
    },
  });
};

export const useDownloadImportTemplate = () => {
  return useMutation({
    mutationFn: () => apiClient.downloadImportTemplate(),
  });
};

// Async Import Job hooks
export const useCreateProductImportJob = () => {
  return useMutation({
    mutationFn: (formData: FormData) => apiClient.createProductImportJob(formData),
  });
};

export const useImportJobProgress = (jobId: string | null, enabled = true) => {
  return useQuery({
    queryKey: ['importJobProgress', jobId],
    queryFn: () => jobId ? apiClient.getImportJobProgress(jobId) : null,
    enabled: enabled && !!jobId,
    refetchInterval: (data) => {
      // Keep polling if job is still processing
      if (data?.state?.data?.job?.status === 'pending' || data?.state?.data?.job?.status === 'processing') {
        return 2000; // Poll every 2 seconds
      }
      return false; // Stop polling when completed or failed
    },
  });
};

// Order hooks
export const useOrders = (params: PaginationParams = {}) => {
  const { user } = useAuth();
  
  return useQuery({
    queryKey: [...queryKeys.orders, params],
    queryFn: () => apiClient.getOrders(params),
    enabled: user?.role !== 'system_admin', // Skip for system_admin
  });
};

export const useOrder = (id: string) => {
  return useQuery({
    queryKey: queryKeys.order(id),
    queryFn: () => apiClient.getOrder(id),
    enabled: !!id,
  });
};

export const useOrdersByCustomer = (customerId: string, params: PaginationParams = {}) => {
  return useQuery({
    queryKey: [...queryKeys.orders, 'customer', customerId, params],
    queryFn: () => apiClient.getOrders({ ...params, search: customerId }),
    enabled: !!customerId,
  });
};

export const useCreateOrder = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (order: Omit<Order, 'id' | 'created_at' | 'updated_at'>) =>
      apiClient.createOrder(order),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.orders });
    },
  });
};

// Recent sales hook for dashboard - OTIMIZADO para reduzir consumo de memória
export const useRecentSales = (limit = 5) => {
  const { user } = useAuth();
  
  return useQuery({
    queryKey: [...queryKeys.orders, 'recent', limit],
    queryFn: () => apiClient.getOrders({ limit }),
    enabled: user?.role !== 'system_admin', // Skip for system_admin
    staleTime: 1000 * 60 * 10, // 10 minutes - TESTE: cache muito longo
    refetchInterval: false, // 🚨 DESABILITADO TEMPORARIAMENTE para teste de memória
    refetchOnWindowFocus: false, // Não recarregar ao focar janela
    refetchOnMount: false, // TESTE: Não recarregar ao montar
    select: (data) => {
      // Transform orders data for dashboard display
      return data.data.map(order => ({
        id: order.id,
        order_id: order.id,
        order_number: order.order_number,
        customer: order.customer_name || 'Cliente não informado',
        customer_id: order.customer_id,
        product: `Pedido #${order.order_number}`,
        value: new Intl.NumberFormat('pt-BR', {
          style: 'currency',
          currency: 'BRL',
        }).format(parseFloat(order.total_amount || '0')),
        time: new Date(order.created_at).toLocaleString('pt-BR', {
          hour: '2-digit',
          minute: '2-digit',
        }),
        status: order.status,
        created_at: order.created_at
      }));
    },
  });
};

export const useUpdateOrder = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, order }: { id: string; order: Partial<Order> }) =>
      apiClient.updateOrder(id, order),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.orders });
      queryClient.invalidateQueries({ queryKey: queryKeys.order(id) });
    },
  });
};

// Order Item hooks
export const useAddOrderItem = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ orderId, item }: { 
      orderId: string; 
      item: { 
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
      } 
    }) => apiClient.addOrderItem(orderId, item),
    onSuccess: (_, { orderId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.orders });
      queryClient.invalidateQueries({ queryKey: queryKeys.order(orderId) });
    },
  });
};

export const useUpdateOrderItem = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ orderId, itemId, item }: { orderId: string; itemId: string; item: { quantity?: number; price?: string } }) =>
      apiClient.updateOrderItem(orderId, itemId, item),
    onSuccess: (_, { orderId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.orders });
      queryClient.invalidateQueries({ queryKey: queryKeys.order(orderId) });
    },
  });
};

export const useRemoveOrderItem = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ orderId, itemId }: { orderId: string; itemId: string }) =>
      apiClient.removeOrderItem(orderId, itemId),
    onSuccess: (_, { orderId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.orders });
      queryClient.invalidateQueries({ queryKey: queryKeys.order(orderId) });
    },
  });
};

// Channel hooks
export const useChannels = (params: PaginationParams = {}) => {
  const { user } = useAuth();
  
  return useQuery({
    queryKey: [...queryKeys.channels, params],
    queryFn: () => apiClient.getChannels(params),
    enabled: user?.role !== 'system_admin', // Skip for system_admin
  });
};

export const useChannel = (id: string) => {
  return useQuery({
    queryKey: queryKeys.channel(id),
    queryFn: () => apiClient.getChannel(id),
    enabled: !!id,
  });
};

export const useCreateChannel = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (channel: Omit<Channel, 'id' | 'tenant_id' | 'created_at' | 'updated_at'>) =>
      apiClient.createChannel(channel),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.channels });
    },
  });
};

export const useUpdateChannel = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, channel }: { id: string; channel: Partial<Channel> }) =>
      apiClient.updateChannel(id, channel),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.channels });
      queryClient.invalidateQueries({ queryKey: queryKeys.channel(id) });
    },
  });
};

export const useDeleteChannel = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: string) => apiClient.deleteChannel(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.channels });
    },
  });
};

export const useMigrateConversations = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ channelId, destinationChannelId }: { channelId: string; destinationChannelId: string }) =>
      apiClient.migrateConversations(channelId, destinationChannelId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.channels });
      queryClient.invalidateQueries({ queryKey: ['conversations'] });
    },
  });
};

// Message hooks
export const useMessages = (conversationId: string, params: PaginationParams = {}) => {
  return useQuery({
    queryKey: [...queryKeys.messages(conversationId), params],
    queryFn: () => apiClient.getMessages(conversationId, params),
    enabled: !!conversationId,
  });
};

export const useCreateMessage = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (message: Omit<Message, 'id' | 'created_at' | 'updated_at'>) =>
      apiClient.createMessage(message),
    onSuccess: (data) => {
      if (data.conversation_id) {
        queryClient.invalidateQueries({ 
          queryKey: queryKeys.messages(data.conversation_id) 
        });
      }
    },
  });
};

export const useSendWhatsAppMessage = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (request: SendMessageRequest) =>
      apiClient.sendWhatsAppMessage(request),
    onSuccess: (data) => {
      if (data.conversation_id) {
        // Invalidate conversation to refetch messages
        queryClient.invalidateQueries({ 
          queryKey: queryKeys.conversation(data.conversation_id) 
        });
        // Also invalidate conversations list to update last message
        queryClient.invalidateQueries({ 
          queryKey: queryKeys.conversations 
        });
        // Invalidate counts since unread status might change
        queryClient.invalidateQueries({ 
          queryKey: queryKeys.conversationCounts 
        });
      }
    },
  });
};

export const useUpdateMessageStatus = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ messageId, status }: { messageId: string; status: string }) =>
      apiClient.updateMessageStatus(messageId, status),
    onSuccess: (data) => {
      if (data.conversation_id) {
        // Invalidate conversation to refetch messages with updated status
        queryClient.invalidateQueries({ 
          queryKey: queryKeys.conversation(data.conversation_id) 
        });
      }
    },
  });
};

export const useCreateNote = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (request: CreateNoteRequest) =>
      apiClient.createNote(request),
    onSuccess: (data) => {
      if (data.conversation_id) {
        // Invalidate conversation to refetch messages including new note
        queryClient.invalidateQueries({ 
          queryKey: queryKeys.conversation(data.conversation_id) 
        });
      }
    },
  });
};

// Message Templates hooks
export const useMessageTemplates = (params: PaginationParams & { category?: string; search?: string } = {}) => {
  return useQuery({
    queryKey: [...queryKeys.messageTemplates, params],
    queryFn: () => apiClient.getMessageTemplates(params),
  });
};

export const useMessageTemplate = (id: string) => {
  return useQuery({
    queryKey: queryKeys.messageTemplate(id),
    queryFn: () => apiClient.getMessageTemplate(id),
    enabled: !!id,
  });
};

export const useCreateMessageTemplate = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (request: CreateTemplateRequest) =>
      apiClient.createMessageTemplate(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ 
        queryKey: queryKeys.messageTemplates 
      });
      queryClient.invalidateQueries({ 
        queryKey: queryKeys.templateCategories 
      });
    },
  });
};

export const useUpdateMessageTemplate = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, request }: { id: string; request: UpdateTemplateRequest }) =>
      apiClient.updateMessageTemplate(id, request),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ 
        queryKey: queryKeys.messageTemplates 
      });
      queryClient.invalidateQueries({ 
        queryKey: queryKeys.messageTemplate(id) 
      });
      queryClient.invalidateQueries({ 
        queryKey: queryKeys.templateCategories 
      });
    },
  });
};

export const useDeleteMessageTemplate = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: string) => apiClient.deleteMessageTemplate(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ 
        queryKey: queryKeys.messageTemplates 
      });
      queryClient.invalidateQueries({ 
        queryKey: queryKeys.templateCategories 
      });
    },
  });
};

export const useTemplateCategories = () => {
  return useQuery({
    queryKey: queryKeys.templateCategories,
    queryFn: () => apiClient.getTemplateCategories(),
  });
};

export const useProcessTemplate = () => {
  return useMutation({
    mutationFn: (request: ProcessTemplateRequest) =>
      apiClient.processTemplate(request),
  });
};

// Tenant hooks (admin only)
export const useTenants = (params: PaginationParams = {}) => {
  return useQuery({
    queryKey: [...queryKeys.tenants, params],
    queryFn: () => apiClient.getTenants(params),
  });
};

export const useTenantsStats = () => {
  return useQuery({
    queryKey: ['tenants', 'stats'],
    queryFn: () => apiClient.getTenantsStats(),
  });
};

export const useTenant = (id: string) => {
  return useQuery({
    queryKey: queryKeys.tenant(id),
    queryFn: () => apiClient.getTenant(id),
    enabled: !!id,
  });
};

export const useCreateTenant = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (tenant: Omit<Tenant, 'id' | 'created_at' | 'updated_at'>) =>
      apiClient.createTenant(tenant),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tenants });
    },
  });
};

export const useUpdateTenant = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, tenant }: { id: string; tenant: Partial<Tenant> }) =>
      apiClient.updateTenant(id, tenant),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tenants });
      queryClient.invalidateQueries({ queryKey: queryKeys.tenant(id) });
    },
  });
};

export const useDeleteTenant = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: string) => apiClient.deleteTenant(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tenants });
    },
  });
};

export const useCreateTenantAdmin = () => {
  return useMutation({
    mutationFn: (data: {
      tenant_id: string;
      name: string;
      email: string;
      phone?: string;
      password: string;
    }) => apiClient.createTenantAdmin(data),
  });
};

// Export hooks (super_admin only)
export const useTenantsForExport = () => {
  return useQuery({
    queryKey: ['tenants-export'],
    queryFn: () => apiClient.getTenantsForExport(),
  });
};

export const useExportTenantProducts = () => {
  return useMutation({
    mutationFn: (tenantId: string) => apiClient.exportTenantProducts(tenantId),
  });
};

export const useTenantUsers = (tenantId: string, params: PaginationParams = {}) => {
  return useQuery({
    queryKey: ['tenant-users', tenantId, params],
    queryFn: () => apiClient.getTenantUsers(tenantId, params),
    enabled: !!tenantId,
  });
};

// Hook for superadmin to get users of a specific tenant
export const useTenantUsersForAdmin = (tenantId: string, params: PaginationParams = {}) => {
  return useQuery({
    queryKey: ['admin-tenant-users', tenantId, params],
    queryFn: () => apiClient.getTenantUsersForAdmin(tenantId, params),
    enabled: !!tenantId,
  });
};

export const useUpdateTenantAdmin = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ userId, data }: { 
      userId: string; 
      data: {
        name: string;
        email: string;
        phone?: string;
        password?: string;
      }
    }) => apiClient.updateTenantAdmin(userId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenant-users'] });
    },
  });
};

// WhatsApp Conversation hooks
export const useConversations = (params: PaginationParams = {}) => {
  return useQuery({
    queryKey: [...queryKeys.conversations, params],
    queryFn: () => apiClient.getConversations(params),
  });
};

export const useConversation = (id: string) => {
  return useQuery({
    queryKey: queryKeys.conversation(id),
    queryFn: () => apiClient.getConversation(id),
    enabled: !!id,
  });
};

export const useMarkConversationAsRead = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: string) => apiClient.markConversationAsRead(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      queryClient.invalidateQueries({ queryKey: queryKeys.conversation(id) });
      queryClient.invalidateQueries({ queryKey: queryKeys.unreadMessages });
      queryClient.invalidateQueries({ queryKey: queryKeys.conversationCounts });
    },
  });
};

export const useArchiveConversation = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: string) => apiClient.archiveConversation(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      queryClient.invalidateQueries({ queryKey: queryKeys.conversation(id) });
      queryClient.invalidateQueries({ queryKey: queryKeys.conversationCounts });
    },
  });
};

export const usePinConversation = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: string) => apiClient.pinConversation(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      queryClient.invalidateQueries({ queryKey: queryKeys.conversation(id) });
    },
  });
};

export const useToggleAIConversation = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: string) => apiClient.toggleAIConversation(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      queryClient.invalidateQueries({ queryKey: queryKeys.conversation(id) });
    },
  });
};

export const useUpdateConversationStatus = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, updates }: { 
      id: string; 
      updates: {
        status?: string;
        priority?: string;
        is_archived?: boolean;
        is_pinned?: boolean;
        assigned_agent_id?: string | null;
      }
    }) => apiClient.updateConversationStatus(id, updates),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      queryClient.invalidateQueries({ queryKey: queryKeys.conversation(id) });
    },
  });
};

export const useAssignConversation = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, agentId }: { id: string; agentId: string | null }) => 
      apiClient.assignConversation(id, agentId),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      queryClient.invalidateQueries({ queryKey: queryKeys.conversation(id) });
      queryClient.invalidateQueries({ queryKey: queryKeys.conversationCounts });
    },
  });
};

export const useAssignUserToConversation = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ conversationId, userId, session }: { 
      conversationId: string; 
      userId: string; 
      session: string; 
    }) => apiClient.assignUserToConversation(conversationId, userId, session),
    onSuccess: (_, { conversationId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      queryClient.invalidateQueries({ queryKey: queryKeys.conversation(conversationId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.conversationCounts });
    },
  });
};

// Analytics and Reports hooks
export const useSalesAnalytics = (params: { 
  start_date?: string; 
  end_date?: string; 
  period?: 'daily' | 'weekly' | 'monthly' 
} = {}) => {
  return useQuery({
    queryKey: [...queryKeys.analytics, params],
    queryFn: () => apiClient.getSalesAnalytics(params),
  });
};

export const useReportsData = (params: { 
  type: 'revenue' | 'orders' | 'customers' | 'products';
  period?: 'daily' | 'weekly' | 'monthly';
  start_date?: string;
  end_date?: string;
}) => {
  return useQuery({
    queryKey: [...queryKeys.reports(params.type), params],
    queryFn: () => apiClient.getReportsData(params),
    enabled: !!params.type,
  });
};

export const useTopProducts = (params: { 
  limit?: number;
  period?: 'daily' | 'weekly' | 'monthly';
  start_date?: string;
  end_date?: string;
} = {}) => {
  return useQuery({
    queryKey: [...queryKeys.topProducts, params],
    queryFn: () => apiClient.getTopProducts(params),
  });
};

// WhatsApp Integration hooks - OTIMIZADO para reduzir consumo de memória
export const useWhatsAppSessionStatus = (channelId?: string, isConnecting = false) => {
  return useQuery({
    queryKey: ['whatsapp-session-status', channelId],
    queryFn: () => apiClient.getWhatsAppSessionStatus(channelId!),
    enabled: !!channelId,
    staleTime: 1000 * 60 * 10, // TESTE: 10 minutos cache muito longo
    refetchInterval: false, // 🚨 DESABILITADO TEMPORARIAMENTE para teste de memória
    refetchOnWindowFocus: false, // Não recarregar ao focar janela
    refetchOnMount: false, // TESTE: Não recarregar ao montar
    gcTime: 1000 * 60 * 2, // TESTE: 2 minutos cache mais curto
  });
};

export const useWhatsAppQR = (channelId?: string) => {
  return useQuery({
    queryKey: ['whatsapp-qr', channelId],
    queryFn: () => apiClient.getWhatsAppQR(channelId!),
    enabled: false, // Only run when manually triggered
    staleTime: 30000, // 30 seconds
    retry: false, // Don't retry on error
  });
};

// Order Statistics hooks
export const useOrderStats = () => {
  return useQuery<OrderStats>({
    queryKey: ['order-stats'],
    queryFn: () => apiClient.getOrderStats(),
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
};

// Dashboard hooks
export const useUnreadMessagesCount = (options: { enabled?: boolean } = {}) => {
  return useQuery({
    queryKey: queryKeys.unreadMessages,
    queryFn: () => apiClient.getUnreadMessagesCount(),
    staleTime: 1000 * 3, // 3 minutos - reduzido para evitar piscadas
    refetchInterval: 1000  * 2, // Refetch every 2 minutes (menos frequente)
    refetchOnWindowFocus: false, // Prevent refetch on window focus
    refetchOnReconnect: false, // Não refetch ao reconectar
    enabled: options.enabled !== false, // Default to true unless explicitly disabled
  });
};

export const useDashboardStats = () => {
  const { user } = useAuth();
  
  return useQuery({
    queryKey: queryKeys.dashboardStats,
    queryFn: () => apiClient.getDashboardStats(),
    staleTime: 1000 * 60 * 5, // 5 minutes
    refetchInterval: 1000 * 60 * 5, // Refetch every 5 minutes
    refetchOnWindowFocus: false, // Prevent refetch on window focus
    enabled: user?.role !== 'system_admin', // Skip for system_admin
  });
};

export const useProductStats = () => {
  const { user } = useAuth();
  
  return useQuery({
    queryKey: ['product-stats'],
    queryFn: () => apiClient.getProductStats(),
    staleTime: 1000 * 60 * 5, // 5 minutes cache for product stats
    refetchInterval: 1000 * 60 * 5, // Refetch every 5 minutes
    enabled: !!user, // Only when user is available - tenant_id comes from JWT for tenant_admin
  });
};


export const useConversationCounts = () => {
  return useQuery({
    queryKey: queryKeys.conversationCounts,
    queryFn: () => apiClient.getConversationCounts(),
    staleTime: 1000 * 60 * 5, // 5 minutos - menos polling
    refetchInterval: 1000 * 60 * 5, // Refetch every 5 minutes
    refetchOnWindowFocus: false, // Prevent refetch on window focus
    refetchOnReconnect: false, // Não refetch ao reconectar
  });
};

// Alert hooks
export const useAlerts = (params: PaginationParams = {}) => {
  return useQuery({
    queryKey: [...queryKeys.alerts, params],
    queryFn: () => apiClient.getAlerts(params),
  });
};

export const useAlert = (id: string) => {
  return useQuery({
    queryKey: queryKeys.alert(id),
    queryFn: () => apiClient.getAlert(id),
    enabled: !!id,
  });
};

export const useCreateAlert = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (alert: CreateAlertRequest) => apiClient.createAlert(alert),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.alerts });
    },
  });
};

export const useUpdateAlert = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, alert }: { id: string; alert: Partial<CreateAlertRequest> }) =>
      apiClient.updateAlert(id, alert),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.alerts });
      queryClient.invalidateQueries({ queryKey: queryKeys.alert(id) });
    },
  });
};

export const useDeleteAlert = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: string) => apiClient.deleteAlert(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.alerts });
    },
  });
};

// Tenant Settings hooks
export const useGlobalAISetting = () => {
  const { user } = useAuth();
  
  return useQuery({
    queryKey: ['tenant-setting', 'ai_global_enabled'],
    queryFn: async () => {
      try {
        const response = await apiClient.getTenantSetting('ai_global_enabled');
        return response.setting?.setting_value === 'true';
      } catch (error) {
        // Default to true if setting doesn't exist
        return true;
      }
    },
    staleTime: 1000 * 60 * 5, // 5 minutes
    enabled: user?.role !== 'system_admin', // Skip for system_admin
  });
};

export const useUpdateGlobalAISetting = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (enabled: boolean) => 
      apiClient.updateTenantSetting('ai_global_enabled', enabled.toString(), 'boolean'),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenant-setting', 'ai_global_enabled'] });
    },
  });
};

// Media upload and sending hooks
export const useSendWhatsAppImage = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: {
      chatId: string;
      file: {
        mimetype: string;
        filename: string;
        url: string;
      };
      caption?: string;
      conversation_id: string;
    }) => apiClient.sendWhatsAppImage(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['conversations'] });
      queryClient.invalidateQueries({ queryKey: ['messages'] });
    },
  });
};

export const useSendWhatsAppDocument = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: {
      chatId: string;
      file: {
        mimetype: string;
        filename: string;
        url: string;
      };
      caption?: string;
      conversation_id: string;
    }) => apiClient.sendWhatsAppDocument(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['conversations'] });
      queryClient.invalidateQueries({ queryKey: ['messages'] });
    },
  });
};

export const useSendWhatsAppAudio = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: {
      chatId: string;
      file: {
        mimetype: string;
        url: string;
      };
      convert?: boolean;
      conversation_id: string;
    }) => apiClient.sendWhatsAppAudio(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['conversations'] });
      queryClient.invalidateQueries({ queryKey: ['messages'] });
    },
  });
};

// Billing hooks
export const useTenantUsage = () => {
  const { user } = useAuth();
  
  return useQuery({
    queryKey: ['tenant-usage'],
    queryFn: () => apiClient.getTenantUsage(),
    staleTime: 1000 * 60 * 5, // 5 minutes
    enabled: user?.role !== 'system_admin', // Skip for system_admin
  });
};

export const useCheckUsageLimit = (resourceType: string, amount?: number) => {
  const { user } = useAuth();
  
  return useQuery({
    queryKey: ['usage-limit', resourceType, amount],
    queryFn: () => apiClient.checkUsageLimit(resourceType, amount),
    enabled: !!resourceType && user?.role !== 'system_admin', // Skip for system_admin
    staleTime: 1000 * 30, // 30 seconds
  });
};

export const useIncrementUsage = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ resourceType, amount }: { resourceType: string; amount: number }) =>
      apiClient.incrementUsage(resourceType, amount),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tenant-usage'] });
    },
  });
};

// Plans hooks
export const usePlans = () => {
  return useQuery({
    queryKey: ['plans'],
    queryFn: () => apiClient.getPlans(),
  });
};

// Admin hooks for tenant stats
export const useTenantCredits = (tenantId: string) => {
  return useQuery({
    queryKey: ['tenant-credits', tenantId],
    queryFn: () => apiClient.getAICreditsForTenant(tenantId),
    enabled: !!tenantId,
  });
};

export const useTenantStats = (tenantId: string) => {
  return useQuery({
    queryKey: ['tenant-stats', tenantId],
    queryFn: () => apiClient.getTenantStats(tenantId),
    enabled: !!tenantId,
  });
};

// Payment Method hooks
export const useActivePaymentMethods = () => {
  return useQuery({
    queryKey: ['payment-methods', 'active'],
    queryFn: () => apiClient.getActivePaymentMethods(),
  });
};
