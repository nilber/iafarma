import { useCallback, useEffect, useRef } from 'react';
import { useAuth } from '@/contexts/AuthContext';
import { useSoundNotification } from '@/contexts/SoundNotificationContext';
import { toast } from 'sonner';
import { useQueryClient } from '@tanstack/react-query';
import { useDebounce } from './useSmartDebounce';
import { monitorSmartInvalidations } from '@/utils/invalidationMonitor';

interface WebSocketMessage {
  type: 'message' | 'order_update' | 'customer_update' | 'notification' | 'whatsapp_message' | 'webhook_notification' | 'human_support_alert' | 'ping' | 'pong' | 'connection';
  data: any;
  timestamp: string;
  alertType?: 'normal' | 'human_support';
}

export function useWebSocket() {
  console.log('useWebSocket hook initialized');
  const { user } = useAuth();
  const queryClient = useQueryClient();
  console.log('useWebSocket - user:', user);
  console.log('useWebSocket - user?.tenant_id:', user?.tenant_id);
  
  const { playNotification, playHumanSupportAlert } = useSoundNotification();
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout>();
  const reconnectAttempts = useRef(0);
  const maxReconnectAttempts = 5;

  // 🚀 INVALIDAÇÕES INTELIGENTES - Debounced para evitar piscadas
  const debouncedInvalidateConversations = useDebounce(() => {
    monitorSmartInvalidations('conversations');
    queryClient.invalidateQueries({ 
      queryKey: ['conversations'],
      refetchType: 'inactive' // Só refaz se a query não está sendo usada ativamente
    });
  }, 2000); // Aguarda 2s sem novas mensagens

  const debouncedInvalidateMessages = useDebounce((conversationId?: string) => {
    monitorSmartInvalidations(`messages-${conversationId || 'all'}`);
    if (conversationId) {
      queryClient.invalidateQueries({ 
        queryKey: ['conversation-messages', conversationId],
        refetchType: 'inactive'
      });
    } else {
      queryClient.invalidateQueries({ 
        queryKey: ['messages'],
        refetchType: 'inactive'
      });
    }
  }, 1500); // Aguarda 1.5s para mensagens específicas

  const debouncedInvalidateUnread = useDebounce(() => {
    monitorSmartInvalidations('unread-messages');
    queryClient.invalidateQueries({ 
      queryKey: ['unread-messages'],
      refetchType: 'inactive'
    });
  }, 3000); // Aguarda 3s para contador (menos crítico)

  const sendMessage = useCallback((message: any) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message));
    } else {
      console.warn('WebSocket is not connected');
    }
  }, []);

  const connect = () => {
    console.log('WebSocket connect() called');
    console.log('WebSocket connect() - user:', user);
    console.log('WebSocket connect() - user?.tenant_id:', user?.tenant_id);
    
    // Don't connect WebSocket for system admins - they don't need real-time message notifications
    if (user?.role === 'system_admin') {
      console.log('WebSocket connect() aborted - system_admin does not need WebSocket connection');
      return;
    }
    
    if (!user?.tenant_id) {
      console.log('WebSocket connect() aborted - no tenant_id available');
      return;
    }

    try {
      console.log('WebSocket connection enabled - connecting to backend');
      
      // Use environment variable for WebSocket host
      const wsBaseUrl = import.meta.env.VITE_API_BASE_WS || 'ws://localhost:8080';
      
      // Add authentication token to the connection URL as query parameter
      const token = localStorage.getItem('access_token');
      if (!token) {
        console.error('No authentication token found');
        return;
      }
      
      const wsUrl = `${wsBaseUrl}/api/v1/ws?token=${encodeURIComponent(token)}`;
      
      wsRef.current = new WebSocket(wsUrl);

      wsRef.current.onopen = () => {
        console.log('WebSocket connected');
        reconnectAttempts.current = 0;
      };

      wsRef.current.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data);
          
          // Handle ping/pong for connection keep-alive
          if (message.type === 'ping') {
            sendMessage({ type: 'pong' });
            return;
          }
          
          if (message.type === 'pong') {
            // Just log that we received a pong response
            console.log('WebSocket pong received');
            return;
          }
          
          handleMessage(message);
        } catch (error) {
          console.error('Error parsing WebSocket message:', error);
        }
      };

      wsRef.current.onclose = (event) => {
        console.log('WebSocket disconnected:', event.code, event.reason);
        
        // Only attempt to reconnect if it wasn't a manual close
        if (event.code !== 1000 && reconnectAttempts.current < maxReconnectAttempts) {
          const delay = Math.pow(2, reconnectAttempts.current) * 1000; // Exponential backoff
          reconnectTimeoutRef.current = setTimeout(() => {
            reconnectAttempts.current++;
            connect();
          }, delay);
        }
      };

      wsRef.current.onerror = (error) => {
        console.error('WebSocket error:', error);
      };

    } catch (error) {
      console.error('Error creating WebSocket connection:', error);
    }
  };

  const handleMessage = (message: WebSocketMessage) => {
    console.log('=== WebSocket Message Received ===');
    console.log('Message type:', message.type);
    console.log('Message data:', message.data);
    console.log('================================');
    
    switch (message.type) {
      case 'whatsapp_message':
        // Nova mensagem do WhatsApp
        console.log('WhatsApp message received via WebSocket:', message);
        
        playNotification();
        toast.success(`🚀 Nova mensagem de WhatsApp!`, {
          description: `De: ${message.data.message?.from || 'WhatsApp'} - ${message.data.message?.body || 'Mensagem recebida'}`,
          duration: 8000,
        });

        // � INVALIDAÇÕES INTELIGENTES - Evitam piscadas com debounce
        console.log('🔄 Starting smart invalidations for conversation:', message.data.conversation_id);
        
        // Invalidar conversações (debounced - aguarda 2s)
        debouncedInvalidateConversations();
        
        // Invalidar mensagens específicas (debounced - aguarda 1.5s)
        if (message.data.conversation_id) {
          debouncedInvalidateMessages(message.data.conversation_id);
        } else {
          debouncedInvalidateMessages();
        }
        
        // Invalidar contador de não lidas (debounced - aguarda 3s)
        debouncedInvalidateUnread();
        break;

      case 'message':
        // Nova mensagem recebida (legacy)
        playNotification();
        toast.info('Nova mensagem recebida', {
          description: `De: ${message.data.customer_name || 'Cliente'}`,
        });
        break;

      case 'order_update':
        // Atualização de pedido
        playNotification();
        toast.info('Pedido atualizado', {
          description: `Pedido ${message.data.order_number} - ${message.data.status}`,
        });
        break;

      case 'customer_update':
        // Atualização de cliente
        toast.success('Cliente atualizado', {
          description: `${message.data.customer_name}`,
        });
        break;

      case 'notification':
        // Notificação geral
        playNotification();
        toast.info(message.data.title || 'Notificação', {
          description: message.data.message,
        });
        break;

      case 'webhook_notification':
        // Nova mensagem via webhook ZapPlus
        console.log('Webhook notification received via WebSocket:', message);
        
        const webhookData = message.data?.data;
        if (webhookData) {
          // Não reproduzir som para atualizações de status de mensagem
          const shouldPlaySound = webhookData.type !== 'message_status_update';
          
          if (shouldPlaySound) {
            playNotification();
          }
          
          // Mostrar toast específico baseado no tipo
          if (webhookData.type === 'message_status_update') {
            // Toast silencioso para atualizações de status
            toast.info(`📋 Status da mensagem atualizado`, {
              description: `Status: ${webhookData.old_status} → ${webhookData.new_status}`,
              duration: 3000,
            });
          } else {
            // Toast normal para outras notificações
            toast.success(`📱 Nova mensagem WhatsApp!`, {
              description: `${webhookData.content?.substring(0, 50) || 'Mensagem recebida'}${webhookData.content?.length > 50 ? '...' : ''}`,
              duration: 8000,
            });
          }

          // Invalidate and refetch conversation lists and messages
          if (webhookData.conversation_id) {
            console.log('🔄 Invalidating queries for conversation:', webhookData.conversation_id);
            
            // Invalidate conversations list
            queryClient.invalidateQueries({ queryKey: ['conversations'] });
            
            // Invalidate messages for this specific conversation
            queryClient.invalidateQueries({ 
              queryKey: ['conversation-messages', webhookData.conversation_id] 
            });
            
            // Invalidate general messages if no specific conversation
            queryClient.invalidateQueries({ queryKey: ['messages'] });
          } else {
            // If no conversation_id, invalidate all conversation-related queries
            console.log('🔄 No conversation_id, invalidating all conversation queries');
            queryClient.invalidateQueries({ queryKey: ['conversations'] });
            queryClient.invalidateQueries({ queryKey: ['messages'] });
          }
        }
        break;

      case 'human_support_alert':
        // Solicitação de atendimento humano
        console.log('Human support alert received via WebSocket:', message);
        
        playHumanSupportAlert();
        toast.error('🆘 Solicitação de Atendimento Humano!', {
          description: `Cliente: ${message.data.customer_name || 'Cliente desconhecido'} - ${message.data.reason || 'Solicitou falar com um atendente'}`,
          duration: 15000, // Mantém o toast por mais tempo para chamadas de atenção
        });
        break;

      default:
        console.log('Unknown message type:', message.type);
    }
  };

  const disconnect = () => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
    }
    
    if (wsRef.current) {
      wsRef.current.close(1000, 'Manual disconnect');
      wsRef.current = null;
    }
  };

  useEffect(() => {
    console.log('useWebSocket useEffect triggered - user?.tenant_id:', user?.tenant_id);
    connect();

    return () => {
      disconnect();
    };
  }, [user?.tenant_id]);

  return {
    isConnected: wsRef.current?.readyState === WebSocket.OPEN,
    sendMessage,
    disconnect
  };
}
