import { useState, useEffect, useRef, useMemo, useCallback, memo } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { MessageSquare, Search, Archive, Pin, MoreVertical, Send, Paperclip, Smile, Loader2, Plus, UserPlus, PinOff, ArchiveRestore, Bot, BotOff, StickyNote, FileText, Edit2, Trash2, Settings, AlertCircle, RotateCcw, User, Clock, MessageCircle, Inbox, UserCheck, Ticket } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { ScrollArea } from "@/components/ui/scroll-area";
import { MessageContent } from "@/components/ui/message-content";
import { TruncatedMessage } from "@/components/TruncatedMessage";
import { MessageStatusIcon } from "@/components/ui/message-status-icon";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { MediaAttachmentDialog } from "@/components/MediaAttachmentDialog";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useConversations, useConversation, useSendWhatsAppMessage, useUpdateMessageStatus, useCreateNote, useMessageTemplates, useCreateMessageTemplate, useUpdateMessageTemplate, useDeleteMessageTemplate, useTemplateCategories, useProcessTemplate, useMarkConversationAsRead, useArchiveConversation, usePinConversation, useToggleAIConversation, useCustomers, useAssignConversation, useConversationCounts, useTenantUsers, useAssignUserToConversation } from "@/lib/api/hooks";
import { MessageTemplate } from "@/lib/api/types";
import { apiClient } from "@/lib/api/client";
import { useQueryClient } from "@tanstack/react-query";
import { format } from "date-fns";
import { ptBR } from "date-fns/locale";
import { toast } from "sonner";
import { useAuth } from "@/contexts/AuthContext";
import { ConversationUsersModal } from "@/components/ConversationUsersModal";
import { UserDelegationDropdown } from "@/components/UserDelegationDropdown";
import { ErrorState } from "@/components/ui/empty-state";

// Memoized search input component to prevent re-renders
const SearchInput = memo(({ 
  value, 
  onChange, 
  isSearching 
}: { 
  value: string;
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  isSearching: boolean;
}) => (
  <div className="relative">
    <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
    <Input 
      placeholder="Buscar conversas..."
      value={value}
      onChange={onChange}
      className="pl-10 pr-10"
      autoComplete="off"
    />
    {isSearching && (
      <Loader2 className="absolute right-3 top-1/2 transform -translate-y-1/2 w-4 h-4 animate-spin text-muted-foreground" />
    )}
  </div>
));

SearchInput.displayName = 'SearchInput';

export default function WhatsAppConversations() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const phoneParam = searchParams.get('phone');
  const customerIdParam = searchParams.get('customer_id');
  const { user } = useAuth();
  
  const [selectedConversationId, setSelectedConversationId] = useState<string | null>(null);
  const [newMessage, setNewMessage] = useState("");
  const [isNewConversationDialogOpen, setIsNewConversationDialogOpen] = useState(false);
  const [customerSearchTerm, setCustomerSearchTerm] = useState("");
  const [conversationSearchInput, setConversationSearchInput] = useState(phoneParam || "");
  const [conversationSearchTerm, setConversationSearchTerm] = useState(phoneParam || "");
  const [filterCategory, setFilterCategory] = useState<'novas' | 'em_atendimento' | 'minhas' | 'arquivadas'>('novas');
  const [isNoteDialogOpen, setIsNoteDialogOpen] = useState(false);
  const [noteContent, setNoteContent] = useState("");
  const [isTemplateDialogOpen, setIsTemplateDialogOpen] = useState(false);
  const [selectedTemplate, setSelectedTemplate] = useState<MessageTemplate | null>(null);
  const [templateVariables, setTemplateVariables] = useState<Record<string, string>>({});
  const [isTemplateFormDialogOpen, setIsTemplateFormDialogOpen] = useState(false);
  const [templateFormMode, setTemplateFormMode] = useState<'create' | 'edit'>('create');
  const [editingTemplate, setEditingTemplate] = useState<MessageTemplate | null>(null);
  const [isMediaAttachmentDialogOpen, setIsMediaAttachmentDialogOpen] = useState(false);
  const [isTicketDialogOpen, setIsTicketDialogOpen] = useState(false);
  const [isUsersModalOpen, setIsUsersModalOpen] = useState(false);
  const [usersModalConversationId, setUsersModalConversationId] = useState<string | null>(null);
  const [usersModalSession, setUsersModalSession] = useState<string>("");
  const [selectedMessageForTicket, setSelectedMessageForTicket] = useState<any>(null);
  const [ticketForm, setTicketForm] = useState({
    title: '',
    description: '',
    priority: 'medium' as 'low' | 'medium' | 'high' | 'urgent',
    category: ''
  });
  const [templateForm, setTemplateForm] = useState({
    title: '',
    content: '',
    category: '',
    description: '',
    variables: [] as string[]
  });
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);

  // Função para abreviar nomes longos na lista de conversas
  const abbreviateName = (fullName: string, maxLength: number = 18): string => {
    if (!fullName || fullName.length <= maxLength) {
      return fullName;
    }

    // Lista de palavras de ligação para remover
    const stopWords = ['de', 'da', 'do', 'das', 'dos', 'e', 'com', 'por', 'para', 'em', 'na', 'no', 'nas', 'nos'];
    
    // Quebrar o nome em palavras
    let nameParts = fullName.trim().split(/\s+/);
    
    // Remover palavras de ligação (mantendo sempre primeira e última palavra)
    if (nameParts.length > 2) {
      const firstWord = nameParts[0];
      const lastWord = nameParts[nameParts.length - 1];
      const middleParts = nameParts.slice(1, -1);
      
      // Filtrar palavras de ligação das partes do meio
      const filteredMiddle = middleParts.filter(part => 
        !stopWords.includes(part.toLowerCase())
      );
      
      nameParts = [firstWord, ...filteredMiddle, lastWord];
    }
    
    // Reconstituir o nome e verificar se ainda é muito longo
    let currentName = nameParts.join(' ');
    
    // Se ainda for muito longo, abreviar do final para o início
    while (currentName.length > maxLength && nameParts.length > 1) {
      // Abreviar a última palavra que não seja a primeira
      for (let i = nameParts.length - 1; i > 0; i--) {
        if (nameParts[i].length > 1) {
          nameParts[i] = nameParts[i].charAt(0);
          break;
        }
      }
      
      currentName = nameParts.join(' ');
      
      // Se todas as palavras já foram abreviadas para 1 char e ainda é longo
      // abreviar palavras do meio para frente (exceto a primeira)
      if (currentName.length > maxLength) {
        const allAbbreviated = nameParts.slice(1).every(part => part.length === 1);
        if (allAbbreviated && nameParts.length > 2) {
          // Remover uma palavra do meio
          nameParts.splice(-2, 1);
          currentName = nameParts.join(' ');
        } else if (allAbbreviated && nameParts.length === 2) {
          // Se só restou 2 palavras e ainda é longo, truncar a primeira
          const firstPart = nameParts[0];
          if (firstPart.length > maxLength - 3) { // -3 para espaço + inicial
            nameParts[0] = firstPart.substring(0, maxLength - 3);
          }
          currentName = nameParts.join(' ');
          break;
        }
      }
    }
    
    return currentName;
  };

  // Initialize search with phone parameter from URL (legacy support)
  useEffect(() => {
    if (phoneParam) {
      setConversationSearchInput(phoneParam);
      setConversationSearchTerm(phoneParam);
    }
  }, [phoneParam]);

  // Handle customer_id parameter to automatically select conversation
  useEffect(() => {
    if (customerIdParam) {
      // Find or create conversation for customer
      const findOrCreateConversation = async () => {
        try {
          const conversation = await apiClient.findOrCreateConversationByCustomer(customerIdParam);
          setSelectedConversationId(conversation.id);
          
          // Clear search to focus on the selected conversation
          setConversationSearchInput("");
          setConversationSearchTerm("");
        } catch (error) {
          console.error('Failed to find/create conversation for customer:', customerIdParam, error);
        }
      };

      findOrCreateConversation();
    }
  }, [customerIdParam]);

  // Stable callback for search input
  const handleSearchInputChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setConversationSearchInput(e.target.value);
  }, []);

  // Debounce search input to avoid too many API calls
  useEffect(() => {
    const timer = setTimeout(() => {
      setConversationSearchTerm(conversationSearchInput);
    }, 500); // Increased to 500ms for better UX

    return () => clearTimeout(timer);
  }, [conversationSearchInput]);

  // React Query
  const queryClient = useQueryClient();

  // Handle customer_id parameter to automatically select conversation
  useEffect(() => {
    if (customerIdParam) {
      console.log('🔍 Customer ID detected in URL:', customerIdParam);
      
      const findOrCreateConversation = async () => {
        try {
          console.log(' Making API call to find/create conversation...');
          const conversation = await apiClient.findOrCreateConversationByCustomer(customerIdParam);
          console.log('✅ Conversation found/created:', conversation.id);
          
          // Invalidate conversations cache to refresh the list
          console.log('🔄 Invalidating conversations cache...');
          queryClient.invalidateQueries({ queryKey: ['conversations'] });
          
          // Wait for cache to update, then select the conversation
          setTimeout(() => {
            console.log('🎯 Setting selected conversation ID...');
            setSelectedConversationId(conversation.id);
            
            // Clear search to focus on the selected conversation
            setConversationSearchInput("");
            setConversationSearchTerm("");
            
            console.log('✅ Conversation selected automatically');
          }, 300);
          
        } catch (error) {
          console.error('❌ Error finding/creating conversation:', error);
        }
      };

      findOrCreateConversation();
    }
  }, [customerIdParam, queryClient]);

  // Memoize query parameters to prevent unnecessary re-renders
  const conversationsQueryParams = useMemo(() => {
    const baseParams = {
      limit: 50,
      search: conversationSearchTerm,
    };

    // Add filter based on category
    switch (filterCategory) {
      case 'arquivadas':
        return { ...baseParams, archived: true };
      case 'minhas':
        return { ...baseParams, assigned_agent_id: user?.id, archived: false };
      case 'em_atendimento':
        return { ...baseParams, archived: false, has_agent: true };
      case 'novas':
      default:
        return { ...baseParams, archived: false, has_agent: false };
    }
  }, [conversationSearchTerm, filterCategory, user?.id]);

  // Fetch conversations with search and archive filter
  const { data: conversationsData, isLoading: conversationsLoading, error: conversationsError } = useConversations(conversationsQueryParams);
  
  // Fetch conversation counts for badges
  const { data: conversationCounts } = useConversationCounts();
  
  // Auto-select conversation when filtering by phone
  useEffect(() => {
    if (phoneParam && conversationsData?.conversations?.length === 1) {
      setSelectedConversationId(conversationsData.conversations[0].id);
    }
  }, [phoneParam, conversationsData]);
  
  // Fetch selected conversation details
  const { data: conversationData, isLoading: conversationLoading } = useConversation(selectedConversationId || '');
  
  // Fetch customers for new conversation
  const { data: customersData, isLoading: customersLoading } = useCustomers({ search: customerSearchTerm });
  
  // Fetch templates for current user
  const { data: templatesData } = useMessageTemplates();
  
  // Mutations
  const sendMessage = useSendWhatsAppMessage();
  const updateMessageStatus = useUpdateMessageStatus();
  const createNote = useCreateNote();
  const processTemplate = useProcessTemplate();
  const markAsRead = useMarkConversationAsRead();
  const archiveConversation = useArchiveConversation();
  const pinConversation = usePinConversation();
  const toggleAIConversation = useToggleAIConversation();
  
  // Template management mutations
  const createTemplate = useCreateMessageTemplate();
  const updateTemplate = useUpdateMessageTemplate();
  const deleteTemplate = useDeleteMessageTemplate();
  const { data: templateCategories } = useTemplateCategories();

  // User delegation hooks - only fetch if user is admin
  const { data: tenantUsersData, isLoading: usersLoading, error: usersError } = useTenantUsers(
    user?.tenant_id || '', 
    { limit: 100 }, // Get enough users for selection
  );
  
  const assignUserToConversation = useAssignUserToConversation();

  // Check if user is admin (tenant_admin or higher role)
  const isAdmin = user?.role === 'tenant_admin' || user?.role === 'admin' || user?.role === 'super_admin';

  // Debug logging
  console.log('Debug delegation conditions:', {
    isAdmin,
    userRole: user?.role,
    tenantId: user?.tenant_id,
    tenantUsersData,
    usersLoading,
    usersError,
    selectedConversationId: selectedConversationId
  });

  // Handle customer selection for new conversation
  const handleSelectCustomer = async (customerId: string) => {
    try {
      // Check if conversation already exists or create a new one
      const conversation = await apiClient.findOrCreateConversationByCustomer(customerId);
      
      // Select the conversation
      setSelectedConversationId(conversation.id);
      
      // Close the dialog
      setIsNewConversationDialogOpen(false);
      setCustomerSearchTerm("");
      
      // Invalidate conversations cache to refresh the list
      queryClient.invalidateQueries({ queryKey: ['conversations'] });
      
      toast.success("Conversa iniciada com sucesso!");
      
    } catch (error) {
      console.error('Error handling customer selection:', error);
      toast.error("Erro ao iniciar conversa");
    }
  };

  const conversations = conversationsData?.conversations || [];
  const selectedConversation = conversationData?.conversation;
  const messages = conversationData?.messages || [];

  // Debug dropdown disable condition
  if (selectedConversation) {
    console.log('Debug dropdown disable condition:', {
      assignUserToConversationPending: assignUserToConversation.isPending,
      channelSession: selectedConversation.channel?.session,
      channelExists: !!selectedConversation.channel,
      channelData: selectedConversation.channel,
      dropdownWillBeDisabled: assignUserToConversation.isPending || !selectedConversation.channel?.session
    });
  }

  // Loading states
  const isSearching = conversationSearchInput !== conversationSearchTerm;
  const isConversationsLoading = conversationsLoading || isSearching;

  // Skeleton component for loading state
  const ConversationSkeleton = () => (
    <div className="flex items-center gap-3 p-3 rounded-lg animate-pulse">
      <div className="w-12 h-12 bg-muted rounded-full"></div>
      <div className="flex-1 space-y-2">
        <div className="h-4 bg-muted rounded w-3/4"></div>
        <div className="h-3 bg-muted rounded w-1/2"></div>
        <div className="h-3 bg-muted rounded w-1/4"></div>
      </div>
    </div>
  );

  // Hook para atribuir conversa
  const assignConversationMutation = useAssignConversation();

  // Função para atribuir conversa ao usuário atual
  const handleAssignConversation = useCallback(async (conversationId: string) => {
    if (!user?.id) return;
    
    try {
      await assignConversationMutation.mutateAsync({
        id: conversationId,
        agentId: user.id,
      });
      
      // Invalida as queries para atualizar os dados na interface
      queryClient.invalidateQueries({ queryKey: ['conversations'] });
      queryClient.invalidateQueries({ queryKey: ['conversation', conversationId] });
      queryClient.invalidateQueries({ queryKey: ['conversation-counts'] });
      
      toast.success("Conversa atribuída com sucesso!");
    } catch (error: any) {
      // Extrai mensagem específica da API se disponível
      const errorMessage = error?.response?.data?.error || error?.message || "Erro ao atribuir conversa";
      toast.error(errorMessage);
      console.error("Erro ao atribuir conversa:", error);
    }
  }, [user?.id, assignConversationMutation, queryClient]);

  // Função para desatribuir conversa (remover agente)
  const handleUnassignConversation = useCallback(async (conversationId: string) => {
    try {
      await assignConversationMutation.mutateAsync({
        id: conversationId,
        agentId: null, // null para desatribuir
      });
      
      // Invalida as queries para atualizar os dados na interface
      queryClient.invalidateQueries({ queryKey: ['conversations'] });
      queryClient.invalidateQueries({ queryKey: ['conversation', conversationId] });
      queryClient.invalidateQueries({ queryKey: ['conversation-counts'] });
      
      toast.success("Conversa desatribuída com sucesso!");
    } catch (error: any) {
      // Extrai mensagem específica da API se disponível
      const errorMessage = error?.response?.data?.error || error?.message || "Erro ao desatribuir conversa";
      toast.error(errorMessage);
      console.error("Erro ao desatribuir conversa:", error);
    }
  }, [assignConversationMutation, queryClient]);

  // Função para delegar conversa a um usuário específico (apenas admins)
  const handleDelegateConversation = useCallback(async (conversationId: string, targetUserId: string) => {
    if (!isAdmin || !selectedConversation?.channel?.session) {
      toast.error("Você não tem permissão para delegar conversas ou sessão WhatsApp não encontrada");
      return;
    }

    try {
      await assignUserToConversation.mutateAsync({
        conversationId,
        userId: targetUserId,
        session: selectedConversation.channel.session,
      });
      
      // Invalida as queries para atualizar os dados na interface
      queryClient.invalidateQueries({ queryKey: ['conversations'] });
      queryClient.invalidateQueries({ queryKey: ['conversation', conversationId] });
      queryClient.invalidateQueries({ queryKey: ['conversation-counts'] });
      
      const targetUser = tenantUsersData?.users.find(u => u.id === targetUserId);
      toast.success(`Conversa delegada para ${targetUser?.name || 'usuário'} com sucesso!`);
    } catch (error: any) {
      // Extrai mensagem específica da API se disponível
      const errorMessage = error?.response?.data?.error || error?.message || "Erro ao delegar conversa";
      toast.error(errorMessage);
      console.error("Erro ao delegar conversa:", error);
    }
  }, [isAdmin, selectedConversation, assignUserToConversation, queryClient, tenantUsersData]);

  // Scroll to bottom when messages change
  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const handleConversationSelect = (conversationId: string) => {
    setSelectedConversationId(conversationId);
    // Mark as read when opening conversation
    markAsRead.mutate(conversationId);
  };

  const handleSendMessage = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newMessage.trim() || !selectedConversationId) return;

    try {
      await sendMessage.mutateAsync({
        conversation_id: selectedConversationId,
        type: 'text',
        content: newMessage.trim(),
      });
      setNewMessage("");
      toast.success("Mensagem enviada!");
    } catch (error) {
      toast.error("Erro ao enviar mensagem");
    }
  };

  const handleResendMessage = async (messageId: string, content: string) => {
    if (!selectedConversationId) return;

    try {
      // First, update the message status to indicate we're retrying
      await updateMessageStatus.mutateAsync({
        messageId: messageId,
        status: 'sending',
      });

      // Try to resend using the sendMessage mutation with resend flag
      await sendMessage.mutateAsync({
        conversation_id: selectedConversationId,
        type: 'text',
        content: content,
        resend_message_id: messageId, // Flag to indicate this is a resend
      });

      // If successful, the backend will update the status automatically
      toast.success("Mensagem reenviada!");
    } catch (error) {
      // If failed, update status back to 'failed'
      await updateMessageStatus.mutateAsync({
        messageId: messageId,
        status: 'failed',
      });
      toast.error("Erro ao reenviar mensagem");
    }
  };

  const handleCreateNote = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!noteContent.trim() || !selectedConversationId) return;

    try {
      await createNote.mutateAsync({
        conversation_id: selectedConversationId,
        content: noteContent.trim(),
      });
      setNoteContent("");
      setIsNoteDialogOpen(false);
      toast.success("Nota adicionada!");
    } catch (error) {
      toast.error("Erro ao adicionar nota");
    }
  };

  const handleSelectTemplate = (template: any) => {
    setSelectedTemplate(template);
    
    // Parse variables from template content
    const variableMatches = template.content.match(/\{\{([^}]+)\}\}/g) || [];
    const variables: Record<string, string> = {};
    
    variableMatches.forEach((match: string) => {
      const variableName = match.replace(/[{}]/g, '');
      variables[variableName] = '';
    });

    // Pre-fill common variables if we have conversation data
    if (conversationData?.conversation?.customer) {
      if (variables['nome_cliente'] !== undefined) {
        variables['nome_cliente'] = conversationData.conversation.customer.name || '';
      }
      if (variables['cliente'] !== undefined) {
        variables['cliente'] = conversationData.conversation.customer.name || '';
      }
    }

    setTemplateVariables(variables);
    setIsTemplateDialogOpen(true);
  };

  const handleUseTemplate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedTemplate || !selectedConversationId) return;

    try {
      const result = await processTemplate.mutateAsync({
        template_id: selectedTemplate.id,
        variables: templateVariables,
      });

      // Set the processed content as the new message
      setNewMessage(result.processed_content);
      setIsTemplateDialogOpen(false);
      setSelectedTemplate(null);
      setTemplateVariables({});
      
      toast.success("Template aplicado!");
    } catch (error) {
      toast.error("Erro ao processar template");
    }
  };

  // Template management functions
  const handleCreateTemplate = () => {
    setTemplateFormMode('create');
    setTemplateForm({
      title: '',
      content: '',
      category: '',
      description: '',
      variables: []
    });
    setEditingTemplate(null);
    setIsTemplateFormDialogOpen(true);
  };

  const handleEditTemplate = (template: MessageTemplate) => {
    setTemplateFormMode('edit');
    setTemplateForm({
      title: template.title,
      content: template.content,
      category: template.category || '',
      description: template.description || '',
      variables: template.variables ? JSON.parse(template.variables) : []
    });
    setEditingTemplate(template);
    setIsTemplateFormDialogOpen(true);
  };

  const handleDeleteTemplate = async (template: MessageTemplate) => {
    if (!confirm(`Tem certeza que deseja excluir o template "${template.title}"?`)) {
      return;
    }

    try {
      await deleteTemplate.mutateAsync(template.id);
      toast.success("Template excluído com sucesso!");
    } catch (error) {
      toast.error("Erro ao excluir template");
    }
  };

  const handleSaveTemplate = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!templateForm.title.trim() || !templateForm.content.trim()) {
      toast.error("Título e conteúdo são obrigatórios");
      return;
    }

    // Extract variables from content
    const variableMatches = templateForm.content.match(/\{\{([^}]+)\}\}/g) || [];
    const extractedVariables = variableMatches.map(match => match.replace(/[{}]/g, ''));

    try {
      const templateData = {
        title: templateForm.title,
        content: templateForm.content,
        category: templateForm.category || undefined,
        description: templateForm.description || undefined,
        variables: extractedVariables
      };

      if (templateFormMode === 'create') {
        await createTemplate.mutateAsync(templateData);
        toast.success("Template criado com sucesso!");
      } else {
        await updateTemplate.mutateAsync({
          id: editingTemplate.id,
          request: templateData
        });
        toast.success("Template atualizado com sucesso!");
      }

      setIsTemplateFormDialogOpen(false);
      setTemplateForm({
        title: '',
        content: '',
        category: '',
        description: '',
        variables: []
      });
    } catch (error) {
      toast.error(`Erro ao ${templateFormMode === 'create' ? 'criar' : 'atualizar'} template`);
    }
  };

  const handleArchiveConversation = async (conversationId: string, currentArchiveState: boolean) => {
    try {
      await archiveConversation.mutateAsync(conversationId);
      if (currentArchiveState) {
        toast.success("Conversa desarquivada!");
      } else {
        toast.success("Conversa arquivada!");
        // If current conversation was archived, clear selection
        if (selectedConversationId === conversationId) {
          setSelectedConversationId(null);
        }
      }
    } catch (error) {
      toast.error("Erro ao alterar estado da conversa");
    }
  };

  const handlePinConversation = async (conversationId: string, currentPinState: boolean) => {
    try {
      await pinConversation.mutateAsync(conversationId);
      if (currentPinState) {
        toast.success("Conversa desfixada!");
      } else {
        toast.success("Conversa fixada!");
      }
    } catch (error) {
      toast.error("Erro ao alterar pin da conversa");
    }
  };

  const handleToggleAI = async (conversationId: string, currentAIState: boolean) => {
    try {
      const result = await toggleAIConversation.mutateAsync(conversationId);
      if (result.ai_enabled) {
        toast.success("IA habilitada para esta conversa!");
      } else {
        toast.success("IA desabilitada para esta conversa!");
      }
    } catch (error) {
      toast.error("Erro ao alterar estado da IA");
    }
  };

  const handleCreateTicketFromMessage = (message: any) => {
    setSelectedMessageForTicket(message);
    setTicketForm({
      title: `Ticket - ${conversationData?.conversation?.customer?.name || 'Cliente'}`,
      description: `Mensagem: ${message.content}\n\nData: ${formatTime(message.created_at)}`,
      priority: 'medium',
      category: ''
    });
    setIsTicketDialogOpen(true);
  };

  const handleCreateTicket = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!ticketForm.title.trim() || !ticketForm.description.trim()) {
      toast.error("Título e descrição são obrigatórios");
      return;
    }

    try {
      const ticketData = {
        title: ticketForm.title,
        description: ticketForm.description,
        priority: ticketForm.priority,
        category_id: ticketForm.category || undefined,
        customer_id: conversationData?.conversation?.customer?.id,
        conversation_id: selectedConversationId,
        channel: 'whatsapp' as const,
        metadata: {
          message_id: selectedMessageForTicket?.id,
          message_content: selectedMessageForTicket?.content,
          message_timestamp: selectedMessageForTicket?.created_at
        }
      };

      await apiClient.createTicket(ticketData);
      toast.success("Ticket criado com sucesso!");
      setIsTicketDialogOpen(false);
      setTicketForm({
        title: '',
        description: '',
        priority: 'medium',
        category: ''
      });
      setSelectedMessageForTicket(null);
    } catch (error) {
      toast.error("Erro ao criar ticket");
    }
  };

  const handleManageUsers = async (conversationId: string) => {
    try {
      // Get conversation details to find the channel and session
      const conversation = await apiClient.getConversation(conversationId);
      
      if (conversation.conversation?.channel?.session) {
        setUsersModalConversationId(conversationId);
        setUsersModalSession(conversation.conversation.channel.session);
        setIsUsersModalOpen(true);
      } else {
        toast.error("Canal não encontrado ou sem sessão configurada");
      }
    } catch (error) {
      console.error("Erro ao obter dados da conversa:", error);
      toast.error("Erro ao carregar dados da conversa");
    }
  };

  // Funções de navegação para os botões do menu
  const handleViewCustomerDetails = useCallback(() => {
    if (selectedConversation?.customer_id) {
      navigate(`/customers/${selectedConversation.customer_id}`);
    } else {
      toast.error("Cliente não encontrado");
    }
  }, [selectedConversation, navigate]);

  const handleViewOrderHistory = useCallback(() => {
    if (selectedConversation?.customer_id) {
      navigate(`/sales/orders?customer_id=${selectedConversation.customer_id}`);
    } else {
      toast.error("Cliente não encontrado");
    }
  }, [selectedConversation, navigate]);

  const formatTime = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffHours = Math.abs(now.getTime() - date.getTime()) / (1000 * 60 * 60);
    
    if (diffHours < 0) {
      return "Agora";
    } else if (diffHours < 24) {
      return format(date, "HH:mm", { locale: ptBR });
    } else {
      return format(date, "dd/MM", { locale: ptBR });
    }
  };

  if (conversationsError) {
    console.error('Conversations error:', conversationsError);
  }

  return (
    <div className="h-[calc(100vh-8rem)] flex gap-6">
      {/* Conversations List */}
      <Card className="w-80 border-0 shadow-custom-md flex flex-col">
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              <MessageSquare className="w-5 h-5 text-whatsapp" />
              Conversas
            </CardTitle>
            <Dialog open={isNewConversationDialogOpen} onOpenChange={setIsNewConversationDialogOpen}>
              <DialogTrigger asChild>
                <Button variant="ghost" size="sm" title="Nova conversa">
                  <UserPlus className="w-4 h-4" />
                </Button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-md max-h-[90vh] overflow-y-auto">
                <DialogHeader>
                  <DialogTitle>Nova Conversa</DialogTitle>
                  <DialogDescription>
                    Selecione um cliente para iniciar uma nova conversa ou reutilizar uma existente.
                  </DialogDescription>
                </DialogHeader>
                <div className="space-y-4">
                  <div className="relative">
                    <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                    <Input
                      placeholder="Buscar cliente..."
                      value={customerSearchTerm}
                      onChange={(e) => setCustomerSearchTerm(e.target.value)}
                      className="pl-10"
                    />
                  </div>
                  <ScrollArea className="h-60">
                    {customersLoading ? (
                      <div className="flex items-center justify-center py-8">
                        <Loader2 className="w-6 h-6 animate-spin" />
                      </div>
                    ) : customersData?.data?.length === 0 ? (
                      <div className="text-center py-8 text-muted-foreground">
                        {customerSearchTerm ? "Nenhum cliente encontrado" : "Nenhum cliente cadastrado"}
                      </div>
                    ) : (
                      <div className="space-y-2">
                        {customersData?.data?.map((customer) => (
                          <div
                            key={customer.id}
                            className="flex items-center gap-3 p-3 rounded-lg hover:bg-muted cursor-pointer"
                            onClick={() => handleSelectCustomer(customer.id)}
                          >
                            <Avatar className="w-10 h-10">
                              <AvatarFallback className="bg-whatsapp/10 text-whatsapp">
                                {customer.name.slice(0, 2).toUpperCase()}
                              </AvatarFallback>
                            </Avatar>
                            <div className="flex-1 min-w-0">
                              <p className="font-medium truncate">{customer.name}</p>
                              <p className="text-sm text-muted-foreground truncate">{customer.phone}</p>
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </ScrollArea>
                </div>
              </DialogContent>
            </Dialog>
          </div>
          <div className="space-y-2">
            <SearchInput 
              value={conversationSearchInput}
              onChange={handleSearchInputChange}
              isSearching={isSearching}
            />
            <div className="space-y-2">
              <div className="flex gap-2 justify-center">
                <Button
                  variant={filterCategory === 'novas' ? "default" : "outline"}
                  size="sm"
                  onClick={() => setFilterCategory('novas')}
                  className="p-2 relative"
                  title="Conversas não atribuídas"
                >
                  <Inbox className="w-4 h-4" />
                  {conversationCounts?.novas !== undefined && conversationCounts.novas > 0 && (
                    <Badge className="absolute -top-2 -right-2 bg-red-500 text-white text-xs min-w-[1.25rem] h-5 flex items-center justify-center p-0">
                      {conversationCounts.novas > 99 ? '99+' : conversationCounts.novas}
                    </Badge>
                  )}
                </Button>
                <Button
                  variant={filterCategory === 'em_atendimento' ? "default" : "outline"}
                  size="sm"
                  onClick={() => setFilterCategory('em_atendimento')}
                  className="p-2 relative"
                  title="Conversas em atendimento"
                >
                  <MessageCircle className="w-4 h-4" />
                  {conversationCounts?.em_atendimento !== undefined && conversationCounts.em_atendimento > 0 && (
                    <Badge className="absolute -top-2 -right-2 bg-blue-500 text-white text-xs min-w-[1.25rem] h-5 flex items-center justify-center p-0">
                      {conversationCounts.em_atendimento > 99 ? '99+' : conversationCounts.em_atendimento}
                    </Badge>
                  )}
                </Button>
                <Button
                  variant={filterCategory === 'minhas' ? "default" : "outline"}
                  size="sm"
                  onClick={() => setFilterCategory('minhas')}
                  className="p-2 relative"
                  title="Minhas conversas"
                >
                  <User className="w-4 h-4" />
                  {conversationCounts?.minhas !== undefined && conversationCounts.minhas > 0 && (
                    <Badge className="absolute -top-2 -right-2 bg-green-500 text-white text-xs min-w-[1.25rem] h-5 flex items-center justify-center p-0">
                      {conversationCounts.minhas > 99 ? '99+' : conversationCounts.minhas}
                    </Badge>
                  )}
                </Button>
                <Button
                  variant={filterCategory === 'arquivadas' ? "default" : "outline"}
                  size="sm"
                  onClick={() => setFilterCategory('arquivadas')}
                  className="p-2 relative"
                  title="Conversas arquivadas"
                >
                  <Archive className="w-4 h-4" />
                  {conversationCounts?.arquivadas !== undefined && conversationCounts.arquivadas > 0 && (
                    <Badge className="absolute -top-2 -right-2 bg-gray-500 text-white text-xs min-w-[1.25rem] h-5 flex items-center justify-center p-0">
                      {conversationCounts.arquivadas > 99 ? '99+' : conversationCounts.arquivadas}
                    </Badge>
                  )}
                </Button>
              </div>
              {/* <div className="flex text-xs text-muted-foreground justify-center gap-4">
                <span className={filterCategory === 'novas' ? 'text-foreground font-medium' : ''}>Novas</span>
                <span className={filterCategory === 'em_atendimento' ? 'text-foreground font-medium' : ''}>Atendimento</span>
                <span className={filterCategory === 'minhas' ? 'text-foreground font-medium' : ''}>Minhas</span>
                <span className={filterCategory === 'arquivadas' ? 'text-foreground font-medium' : ''}>Arquivadas</span>
              </div> */}
            </div>
          </div>
        </CardHeader>
        <CardContent className="flex-1 p-0 overflow-hidden">
          <ScrollArea className="h-[calc(100vh-20rem)]">
            <div className="space-y-1 p-3">
              {conversationsError ? (
                <ErrorState
                  title="Erro ao carregar conversas"
                  message={conversationsError?.message}
                  onRetry={() => window.location.reload()}
                />
              ) : isConversationsLoading && conversations.length === 0 ? (
                // Show skeleton loading for initial load
                <>
                  {Array.from({ length: 3 }).map((_, index) => (
                    <ConversationSkeleton key={index} />
                  ))}
                </>
              ) : (
                <>
                  {/* Show existing conversations with optional overlay for search loading */}
                  <div className={`transition-opacity duration-200 ${isSearching ? 'opacity-60' : 'opacity-100'}`}>
                    {conversations.map((conv) => (
                      <div 
                        key={conv.id}
                        className={`group flex items-center gap-3 p-3 rounded-lg cursor-pointer transition-colors ${
                          selectedConversationId === conv.id 
                            ? 'bg-whatsapp/10 border border-whatsapp/20' 
                            : 'hover:bg-accent/50'
                        }`}
                      >
                  <div 
                    className="flex items-center gap-3 flex-1 min-w-0"
                    onClick={() => handleConversationSelect(conv.id)}
                  >
                    <div className="relative">
                      <Avatar className="w-12 h-12">
                        <AvatarFallback className="bg-whatsapp text-whatsapp-foreground">
                          {conv.customer?.name ? conv.customer.name.split(' ').map(n => n[0]).join('') : '??'}
                        </AvatarFallback>
                      </Avatar>
                      {conv.status === 'active' && (
                        <div className="absolute -bottom-1 -right-1 w-4 h-4 bg-success rounded-full border-2 border-background"></div>
                      )}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2 flex-1 min-w-0">
                          <p className="font-medium text-foreground truncate">
                            {abbreviateName(conv.customer?.name || 'Unknown Customer')}
                          </p>
                          {conv.assigned_agent_id && (
                            <UserCheck className="w-3 h-3 text-blue-500" />
                          )}
                          {conv.is_pinned && (
                            <Pin className="w-3 h-3 text-yellow-500 fill-yellow-500" />
                          )}
                        </div>
                        <span className="text-xs text-muted-foreground flex-shrink-0">{formatTime(conv.last_message_at)}</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <p className="text-sm text-muted-foreground truncate">
                          {typeof conv.last_message === 'string' ? conv.last_message : conv.last_message?.content || 'Sem mensagens'}
                        </p>
                        {conv.unread_count > 0 && (
                          <Badge className="bg-whatsapp text-whatsapp-foreground ml-2">
                            {conv.unread_count}
                          </Badge>
                        )}
                      </div>
                      <p className="text-xs text-muted-foreground">{conv.customer?.phone || 'No phone'}</p>
                    </div>
                  </div>
                  <div className="opacity-0 group-hover:opacity-100 transition-opacity">
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="sm" onClick={(e) => e.stopPropagation()}>
                          <MoreVertical className="w-4 h-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        {!conv.assigned_agent_id && filterCategory !== 'minhas' && (
                          <DropdownMenuItem onClick={(e) => {
                            e.stopPropagation();
                            handleAssignConversation(conv.id);
                          }}>
                            <UserCheck className="w-4 h-4 mr-2" />
                            Atribuir a mim
                          </DropdownMenuItem>
                        )}
                        <DropdownMenuItem onClick={(e) => {
                          e.stopPropagation();
                          handlePinConversation(conv.id, conv.is_pinned);
                        }}>
                          {conv.is_pinned ? (
                            <>
                              <PinOff className="w-4 h-4 mr-2" />
                              Desfixar
                            </>
                          ) : (
                            <>
                              <Pin className="w-4 h-4 mr-2" />
                              Fixar
                            </>
                          )}
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={(e) => {
                          e.stopPropagation();
                          handleArchiveConversation(conv.id, conv.is_archived);
                        }}>
                          {conv.is_archived ? (
                            <>
                              <ArchiveRestore className="w-4 h-4 mr-2" />
                              Desarquivar
                            </>
                          ) : (
                            <>
                              <Archive className="w-4 h-4 mr-2" />
                              Arquivar
                            </>
                          )}
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={(e) => {
                          e.stopPropagation();
                          handleManageUsers(conv.id);
                        }}>
                          <User className="w-4 h-4 mr-2" />
                          Gerenciar Usuários
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                </div>
              ))}
                  </div>
                  
                  {/* Show skeleton loading during search if we have existing conversations */}
                  {isSearching && conversations.length > 0 && (
                    <div className="mt-2">
                      <ConversationSkeleton />
                    </div>
                  )}
                </>
              )}
            </div>
          </ScrollArea>
        </CardContent>
      </Card>

      {/* Chat Area */}
      <Card className="flex-1 border-0 shadow-custom-md flex flex-col">
        {!selectedConversationId ? (
          <div className="flex-1 flex items-center justify-center">
            <div className="text-center">
              <MessageSquare className="w-16 h-16 text-muted-foreground mx-auto mb-4" />
              <h3 className="text-lg font-medium text-foreground mb-2">Selecione uma conversa</h3>
              <p className="text-muted-foreground">Escolha uma conversa para começar a trocar mensagens</p>
            </div>
          </div>
        ) : (
          <>
            {/* Chat Header */}
            {selectedConversation && (
              <CardHeader className="pb-3 border-b">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <Avatar className="w-10 h-10">
                      <AvatarFallback className="bg-whatsapp text-whatsapp-foreground">
                        {selectedConversation.customer?.name ? selectedConversation.customer.name.split(' ').map(n => n[0]).join('') : '??'}
                      </AvatarFallback>
                    </Avatar>
                    <div>
                      <div className="flex items-center gap-2">
                        <p className="font-medium text-foreground">{selectedConversation.customer?.name || 'Unknown Customer'}</p>
                        {selectedConversation.is_pinned && (
                          <Pin className="w-3 h-3 text-yellow-500 fill-yellow-500" />
                        )}
                      </div>
                      <p className="text-sm text-muted-foreground">{selectedConversation.customer?.phone || 'No phone'}</p>
                      {selectedConversation.channel && (
                        <div className="flex items-center gap-1 mt-1">
                          <MessageSquare className="w-3 h-3 text-whatsapp" />
                          <span className="text-xs text-whatsapp font-medium">
                            Canal: {selectedConversation.channel.name}
                          </span>
                        </div>
                      )}
                      {/* Informações de atribuição */}
                      <div className="flex items-center gap-1 mt-1">
                        {selectedConversation.assigned_agent_id ? (
                          <div className="flex items-center gap-1">
                            <UserCheck className="w-3 h-3 text-blue-500" />
                            <span className="text-xs text-blue-600 font-medium">
                              {selectedConversation.assigned_agent_id === user?.id 
                                ? 'Atribuída a você' 
                                : selectedConversation.assigned_agent?.name 
                                  ? `Atribuída a ${selectedConversation.assigned_agent.name}`
                                  : `Atribuída (ID: ${selectedConversation.assigned_agent_id.substring(0, 8)}...)`
                              }
                            </span>
                          </div>
                        ) : (
                          <div className="flex items-center gap-1">
                            <Clock className="w-3 h-3 text-orange-500" />
                            <span className="text-xs text-orange-600 font-medium">Aguardando atribuição</span>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    {/* Botão de atribuição */}
                    {!selectedConversation.assigned_agent_id ? (
                      <Button 
                        variant="outline" 
                        size="sm"
                        onClick={() => handleAssignConversation(selectedConversation.id)}
                        title="Atribuir conversa a mim"
                        className="border-blue-200 text-blue-600 hover:bg-blue-50"
                      >
                        <UserCheck className="w-4 h-4 mr-1" />
                        Atribuir a mim
                      </Button>
                    ) : selectedConversation.assigned_agent_id !== user?.id ? (
                      <Button 
                        variant="outline" 
                        size="sm"
                        onClick={() => handleAssignConversation(selectedConversation.id)}
                        title="Reatribuir conversa a mim"
                        className="border-orange-200 text-orange-600 hover:bg-orange-50"
                      >
                        <UserCheck className="w-4 h-4 mr-1" />
                        Reatribuir
                      </Button>
                    ) : (
                      <Button 
                        variant="outline" 
                        size="sm"
                        onClick={() => {
                          // Aqui poderia abrir um modal para desatribuir ou trocar
                          toast.info("Conversa atribuída a você");
                        }}
                        title="Conversa atribuída a você"
                        className="border-green-200 text-green-600 hover:bg-green-50"
                        disabled
                      >
                        <UserCheck className="w-4 h-4 mr-1" />
                        Sua conversa
                      </Button>
                    )}
                    
                    {/* Dropdown de delegação - apenas para admins */}
                    {isAdmin && tenantUsersData?.users && tenantUsersData.users.length > 0 && (
                      <UserDelegationDropdown
                        users={tenantUsersData.users}
                        isLoading={usersLoading}
                        onSelectUser={(userId) => handleDelegateConversation(selectedConversation.id, userId)}
                        disabled={assignUserToConversation.isPending || !selectedConversation.channel?.session}
                        currentUserId={user?.id}
                      />
                    )}
                    
                    <Button 
                      variant="ghost" 
                      size="sm"
                      onClick={() => handlePinConversation(selectedConversation.id, selectedConversation.is_pinned)}
                      title={selectedConversation.is_pinned ? "Desfixar conversa" : "Fixar conversa"}
                    >
                      {selectedConversation.is_pinned ? (
                        <PinOff className="w-4 h-4" />
                      ) : (
                        <Pin className="w-4 h-4" />
                      )}
                    </Button>
                    <Button 
                      variant="ghost" 
                      size="sm"
                      onClick={() => handleArchiveConversation(selectedConversation.id, selectedConversation.is_archived)}
                      title={selectedConversation.is_archived ? "Desarquivar conversa" : "Arquivar conversa"}
                    >
                      {selectedConversation.is_archived ? (
                        <ArchiveRestore className="w-4 h-4" />
                      ) : (
                        <Archive className="w-4 h-4" />
                      )}
                    </Button>
                    <Button 
                      variant="ghost" 
                      size="sm"
                      onClick={() => handleToggleAI(selectedConversation.id, selectedConversation.ai_enabled)}
                      title={selectedConversation.ai_enabled ? "Desabilitar IA" : "Habilitar IA"}
                      className={`transition-all duration-200 ${
                        selectedConversation.ai_enabled 
                          ? "text-green-600 hover:text-green-700 hover:bg-green-50" 
                          : "text-red-500 hover:text-red-600 hover:bg-red-50"
                      }`}
                    >
                      {selectedConversation.ai_enabled ? (
                        <Bot className="w-4 h-4 animate-pulse" />
                      ) : (
                        <BotOff className="w-4 h-4" />
                      )}
                    </Button>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="sm">
                          <MoreVertical className="w-4 h-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        {selectedConversation.assigned_agent_id && (
                          <DropdownMenuItem onClick={() => handleUnassignConversation(selectedConversation.id)}>
                            <UserCheck className="w-4 h-4 mr-2" />
                            Desatribuir conversa
                          </DropdownMenuItem>
                        )}
                        <DropdownMenuItem onClick={handleViewCustomerDetails}>
                          <User className="w-4 h-4 mr-2" />
                          Detalhes do Cliente
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={handleViewOrderHistory}>
                          <FileText className="w-4 h-4 mr-2" />
                          Histórico de Compras
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                </div>
              </CardHeader>
            )}

            {/* Messages */}
            <CardContent className="flex-1 p-0 overflow-hidden">
              {conversationLoading ? (
                <div className="flex items-center justify-center h-full">
                  <Loader2 className="w-6 h-6 animate-spin" />
                </div>
              ) : (
                <ScrollArea className="h-[calc(100vh-20rem)] p-4">
                  <div className="space-y-4">
                    {messages.map((message) => {
                      // Render notes with special styling
                      if (message.is_note || message.direction === 'note') {
                        return (
                          <div key={message.id} className="flex justify-center">
                            <div className="max-w-md">
                              <div className="bg-yellow-100 border border-yellow-300 px-4 py-2 rounded-lg">
                                <div className="flex items-start gap-2">
                                  <StickyNote className="w-4 h-4 text-yellow-600 mt-0.5 flex-shrink-0" />
                                  <div className="flex-1">
                                    <p className="text-xs text-yellow-700 font-medium mb-1">
                                      Nota interna - {message.user_name || 'Usuário'}
                                    </p>
                                    <p className="text-sm text-yellow-800">{message.content}</p>
                                    <p className="text-xs text-yellow-600 mt-1">
                                      {formatTime(message.created_at)}
                                    </p>
                                  </div>
                                </div>
                              </div>
                            </div>
                          </div>
                        );
                      }

                      // Render regular messages
                      return (
                        <div 
                          key={message.id}
                          className={`group flex ${message.direction === 'in' ? 'justify-start' : 'justify-end'} relative`}
                        >
                          <div className={`max-w-xs lg:max-w-md ${
                            message.direction === 'in' 
                              ? 'order-1' 
                              : 'order-2'
                          }`}>
                            {/* Show sender name only for outgoing messages (tenant side) */}
                            {message.direction === 'out' && (
                              <p className="text-xs text-muted-foreground mb-1 text-right">
                                {message.user_name || 'Assistente IA'}
                              </p>
                            )}
                            <div className={`px-4 py-2 rounded-lg ${
                              message.direction === 'in' 
                                ? 'bg-muted text-foreground' 
                                : 'bg-whatsapp text-whatsapp-foreground'
                            }`}>
                              <TruncatedMessage 
                                content={message.content} 
                                type={message.type}
                                mediaUrl={message.media_url}
                                mediaType={message.media_type}
                                filename={message.filename}
                                className={`text-sm ${
                                  message.direction === 'in' 
                                    ? 'prose-gray' 
                                    : 'prose-white'
                                }`}
                              />
                              <div className="flex items-center justify-between mt-1">
                                <p className={`text-xs ${
                                  message.direction === 'in' 
                                    ? 'text-muted-foreground'
                                    : 'text-whatsapp-foreground/70' 
                                }`}>
                                  {formatTime(message.created_at)}
                                </p>
                                
                                {/* Status indicator para mensagens de saída */}
                                {message.direction === 'out' && (
                                  <div className="flex items-center gap-1 ml-2">
                                    {message.status === 'failed' ? (
                                      // Para mensagens com falha, mostrar botão de reenvio
                                      <>
                                        <AlertCircle className="w-3.5 h-3.5 text-red-500" />
                                        <Button
                                          type="button"
                                          variant="ghost"
                                          size="sm"
                                          className="h-auto p-1 text-xs text-red-600 hover:text-red-800"
                                          onClick={() => handleResendMessage(message.id, message.content || '')}
                                          disabled={sendMessage.isPending}
                                        >
                                          <RotateCcw className="w-3 h-3 mr-1" />
                                          Reenviar
                                        </Button>
                                      </>
                                    ) : (
                                      // Para outras mensagens, mostrar indicador de status WhatsApp
                                      <MessageStatusIcon status={message.status}  />
                                    )}
                                  </div>
                                )}
                              </div>
                            </div>
                          </div>
                          
                          {/* Dropdown Menu - appear on hover - only for incoming messages (customer messages) */}
                          {message.direction === 'in' && (
                            <div className="absolute top-1 right-2 opacity-0 group-hover:opacity-100 transition-opacity">
                              <DropdownMenu>
                                <DropdownMenuTrigger asChild>
                                  <Button 
                                    variant="ghost" 
                                    size="sm"
                                    className="h-6 w-6 p-0 bg-white/80 hover:bg-white border shadow-sm"
                                    onClick={(e) => e.stopPropagation()}
                                  >
                                    <MoreVertical className="w-3 h-3" />
                                  </Button>
                                </DropdownMenuTrigger>
                                <DropdownMenuContent align="end" className="w-48">
                                  <DropdownMenuItem 
                                    onClick={(e) => {
                                      e.stopPropagation();
                                      handleCreateTicketFromMessage(message);
                                    }}
                                  >
                                    <Ticket className="w-4 h-4 mr-2" />
                                    Criar Ticket
                                  </DropdownMenuItem>
                                </DropdownMenuContent>
                              </DropdownMenu>
                            </div>
                          )}
                        </div>
                      );
                    })}
                    <div ref={messagesEndRef} />
                  </div>
                </ScrollArea>
              )}
            </CardContent>

            {/* Message Input */}
            {selectedConversationId && (
              <div className="p-4 border-t">
                <form onSubmit={handleSendMessage} className="flex items-center gap-3">
                  <Button 
                    type="button" 
                    variant="ghost" 
                    size="sm"
                    onClick={() => setIsMediaAttachmentDialogOpen(true)}
                  >
                    <Paperclip className="w-4 h-4" />
                  </Button>
                  <div className="flex-1 relative">
                    <Input 
                      placeholder="Digite sua mensagem..."
                      value={newMessage}
                      onChange={(e) => setNewMessage(e.target.value)}
                      className="pr-10"
                      disabled={sendMessage.isPending}
                    />
                    <Button 
                      type="button"
                      variant="ghost" 
                      size="sm" 
                      className="absolute right-1 top-1/2 transform -translate-y-1/2"
                    >
                      <Smile className="w-4 h-4" />
                    </Button>
                  </div>
                  
                  {/* Template Dialog */}
                  <Dialog open={isTemplateDialogOpen} onOpenChange={setIsTemplateDialogOpen}>
                    <DialogTrigger asChild>
                      <Button 
                        type="button"
                        variant="outline" 
                        size="sm"
                        className="text-blue-600 border-blue-300 hover:bg-blue-50"
                      >
                        <FileText className="w-4 h-4" />
                      </Button>
                    </DialogTrigger>
                    <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
                      <DialogHeader>
                        <div className="flex items-center justify-between">
                          <div>
                            <DialogTitle>Selecionar Template</DialogTitle>
                            <DialogDescription>
                              Escolha um template e preencha as variáveis para usar na conversa.
                            </DialogDescription>
                          </div>
                          <Button
                            onClick={handleCreateTemplate}
                            size="sm"
                            className="bg-blue-600 hover:bg-blue-700"
                          >
                            <Plus className="w-4 h-4 mr-2" />
                            Novo Template
                          </Button>
                        </div>
                      </DialogHeader>
                      
                      {!selectedTemplate ? (
                        <div className="space-y-4">
                          <ScrollArea className="h-[350px]">
                            <div className="space-y-2">
                              {templatesData?.map((template) => (
                                <div
                                  key={template.id}
                                  className="p-3 border rounded-lg hover:bg-gray-50 group"
                                >
                                  <div className="flex justify-between items-start">
                                    <div 
                                      className="flex-1 cursor-pointer"
                                      onClick={() => handleSelectTemplate(template)}
                                    >
                                      <h4 className="font-medium">{template.title}</h4>
                                      {template.description && (
                                        <p className="text-sm text-gray-600 mt-1">{template.description}</p>
                                      )}
                                      <p className="text-sm text-gray-500 mt-2 line-clamp-2">
                                        {template.content}
                                      </p>
                                      {template.category && (
                                        <Badge variant="secondary" className="mt-2">
                                          {template.category}
                                        </Badge>
                                      )}
                                    </div>
                                    <div className="flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                                      <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={(e) => {
                                          e.stopPropagation();
                                          handleEditTemplate(template);
                                        }}
                                        className="h-8 w-8 p-0"
                                      >
                                        <Edit2 className="w-4 h-4" />
                                      </Button>
                                      <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={(e) => {
                                          e.stopPropagation();
                                          handleDeleteTemplate(template);
                                        }}
                                        className="h-8 w-8 p-0 text-red-500 hover:text-red-700"
                                      >
                                        <Trash2 className="w-4 h-4" />
                                      </Button>
                                    </div>
                                  </div>
                                </div>
                              ))}
                              {(!templatesData || templatesData.length === 0) && (
                                <div className="text-center py-8 text-gray-500">
                                  <FileText className="w-12 h-12 mx-auto mb-4 text-gray-300" />
                                  <p className="text-lg font-medium mb-2">Nenhum template encontrado</p>
                                  <p className="text-sm mb-4">Crie seu primeiro template para agilizar o atendimento</p>
                                  <Button
                                    onClick={handleCreateTemplate}
                                    className="bg-blue-600 hover:bg-blue-700"
                                  >
                                    <Plus className="w-4 h-4 mr-2" />
                                    Criar Primeiro Template
                                  </Button>
                                </div>
                              )}
                            </div>
                          </ScrollArea>
                        </div>
                      ) : (
                        <form onSubmit={handleUseTemplate} className="space-y-4">
                          <div className="p-3 bg-gray-50 rounded-lg">
                            <h4 className="font-medium">{selectedTemplate.title}</h4>
                            <p className="text-sm text-gray-600 mt-1">{selectedTemplate.content}</p>
                          </div>
                          
                          {Object.keys(templateVariables).length > 0 && (
                            <div className="space-y-3">
                              <h5 className="font-medium">Preencha as variáveis:</h5>
                              {Object.entries(templateVariables).map(([variable, value]) => (
                                <div key={variable}>
                                  <label className="block text-sm font-medium mb-1">
                                    {variable.replace(/_/g, ' ').toUpperCase()}
                                  </label>
                                  <Input
                                    value={value}
                                    onChange={(e) => setTemplateVariables(prev => ({
                                      ...prev,
                                      [variable]: e.target.value
                                    }))}
                                    placeholder={`Digite ${variable.replace(/_/g, ' ')}`}
                                  />
                                </div>
                              ))}
                            </div>
                          )}
                          
                          <div className="flex justify-end gap-2">
                            <Button
                              type="button"
                              variant="outline"
                              onClick={() => {
                                setSelectedTemplate(null);
                                setTemplateVariables({});
                                setIsTemplateDialogOpen(false);
                              }}
                              disabled={processTemplate.isPending}
                            >
                              Voltar
                            </Button>
                            <Button
                              type="submit"
                              disabled={processTemplate.isPending}
                            >
                              {processTemplate.isPending ? (
                                <Loader2 className="w-4 h-4 animate-spin" />
                              ) : (
                                'Usar Template'
                              )}
                            </Button>
                          </div>
                        </form>
                      )}
                    </DialogContent>
                  </Dialog>

                  <Dialog open={isNoteDialogOpen} onOpenChange={setIsNoteDialogOpen}>
                    <DialogTrigger asChild>
                      <Button 
                        type="button"
                        variant="outline" 
                        size="sm"
                        className="text-yellow-600 border-yellow-300 hover:bg-yellow-50"
                      >
                        <StickyNote className="w-4 h-4" />
                      </Button>
                    </DialogTrigger>
                    <DialogContent>
                      <DialogHeader>
                        <DialogTitle>Adicionar Nota Interna</DialogTitle>
                        <DialogDescription>
                          Esta nota será visível apenas para os usuários do sistema e não será enviada ao cliente.
                        </DialogDescription>
                      </DialogHeader>
                      <form onSubmit={handleCreateNote} className="space-y-4">
                        <div>
                          <textarea
                            value={noteContent}
                            onChange={(e) => setNoteContent(e.target.value)}
                            placeholder="Digite sua nota..."
                            className="w-full min-h-[100px] p-3 border rounded-lg resize-none focus:outline-none focus:ring-2 focus:ring-yellow-500"
                            disabled={createNote.isPending}
                          />
                        </div>
                        <div className="flex justify-end gap-2">
                          <Button
                            type="button"
                            variant="outline"
                            onClick={() => {
                              setIsNoteDialogOpen(false);
                              setNoteContent("");
                            }}
                            disabled={createNote.isPending}
                          >
                            Cancelar
                          </Button>
                          <Button
                            type="submit"
                            className="bg-yellow-600 hover:bg-yellow-700"
                            disabled={!noteContent.trim() || createNote.isPending}
                          >
                            {createNote.isPending ? (
                              <Loader2 className="w-4 h-4 animate-spin" />
                            ) : (
                              'Adicionar Nota'
                            )}
                          </Button>
                        </div>
                      </form>
                    </DialogContent>
                  </Dialog>
                  <Button 
                    type="submit" 
                    size="sm" 
                    className="bg-whatsapp hover:bg-whatsapp-dark"
                    disabled={!newMessage.trim() || sendMessage.isPending}
                  >
                    {sendMessage.isPending ? (
                      <Loader2 className="w-4 h-4 animate-spin" />
                    ) : (
                      <Send className="w-4 h-4" />
                    )}
                  </Button>
                </form>
              </div>
            )}
          </>
        )}
      </Card>

      {/* Template Form Dialog */}
      <Dialog open={isTemplateFormDialogOpen} onOpenChange={setIsTemplateFormDialogOpen}>
        <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>
              {templateFormMode === 'create' ? 'Criar Novo Template' : 'Editar Template'}
            </DialogTitle>
            <DialogDescription>
              {templateFormMode === 'create' 
                ? 'Crie um novo template para agilizar suas respostas' 
                : 'Edite as informações do template'
              }
            </DialogDescription>
          </DialogHeader>
          
          <form onSubmit={handleSaveTemplate} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="template-title">Título *</Label>
                <Input
                  id="template-title"
                  placeholder="Ex: Saudação inicial"
                  value={templateForm.title}
                  onChange={(e) => setTemplateForm(prev => ({ ...prev, title: e.target.value }))}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="template-category">Categoria</Label>
                <Input
                  id="template-category"
                  placeholder="Ex: Atendimento, Vendas"
                  value={templateForm.category}
                  onChange={(e) => setTemplateForm(prev => ({ ...prev, category: e.target.value }))}
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="template-description">Descrição</Label>
              <Input
                id="template-description"
                placeholder="Breve descrição do template"
                value={templateForm.description}
                onChange={(e) => setTemplateForm(prev => ({ ...prev, description: e.target.value }))}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="template-content">Conteúdo *</Label>
              <Textarea
                id="template-content"
                placeholder="Digite o conteúdo do template. Use {{variavel}} para criar variáveis dinâmicas."
                value={templateForm.content}
                onChange={(e) => setTemplateForm(prev => ({ ...prev, content: e.target.value }))}
                className="min-h-[120px]"
                required
              />
              <p className="text-xs text-muted-foreground">
                💡 Dica: Use {`{{nome_cliente}}`} ou {`{{produto}}`} para criar variáveis que serão preenchidas ao usar o template
              </p>
            </div>

            {templateForm.content && (
              <div className="space-y-2">
                <Label>Variáveis Detectadas</Label>
                <div className="flex flex-wrap gap-2">
                  {(templateForm.content.match(/\{\{([^}]+)\}\}/g) || []).map((variable, index) => (
                    <Badge key={index} variant="secondary">
                      {variable}
                    </Badge>
                  ))}
                </div>
              </div>
            )}

            <div className="flex justify-end gap-3 pt-4">
              <Button
                type="button"
                variant="outline"
                onClick={() => setIsTemplateFormDialogOpen(false)}
                disabled={createTemplate.isPending || updateTemplate.isPending}
              >
                Cancelar
              </Button>
              <Button
                type="submit"
                disabled={createTemplate.isPending || updateTemplate.isPending}
                className="bg-blue-600 hover:bg-blue-700"
              >
                {(createTemplate.isPending || updateTemplate.isPending) ? (
                  <Loader2 className="w-4 h-4 animate-spin mr-2" />
                ) : templateFormMode === 'create' ? (
                  <Plus className="w-4 h-4 mr-2" />
                ) : (
                  <Settings className="w-4 h-4 mr-2" />
                )}
                {templateFormMode === 'create' ? 'Criar Template' : 'Salvar Alterações'}
              </Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>

      {/* Media Attachment Dialog */}
      <MediaAttachmentDialog
        isOpen={isMediaAttachmentDialogOpen}
        onClose={() => setIsMediaAttachmentDialogOpen(false)}
        conversationId={selectedConversationId || ''}
        chatId={selectedConversationId || ''}
      />

      {/* Ticket Creation Dialog */}
      <Dialog open={isTicketDialogOpen} onOpenChange={setIsTicketDialogOpen}>
        <DialogContent className="sm:max-w-md max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Criar Ticket da Mensagem</DialogTitle>
            <DialogDescription>
              Criar um ticket de suporte baseado na mensagem selecionada.
            </DialogDescription>
          </DialogHeader>
          
          <form onSubmit={handleCreateTicket} className="space-y-4">
            <div className="space-y-2">
              <label htmlFor="ticket-title" className="text-sm font-medium">
                Título do Ticket
              </label>
              <Input
                id="ticket-title"
                placeholder="Digite o título do ticket..."
                value={ticketForm.title}
                onChange={(e) => setTicketForm({ ...ticketForm, title: e.target.value })}
                required
              />
            </div>

            <div className="space-y-2">
              <label htmlFor="ticket-description" className="text-sm font-medium">
                Descrição
              </label>
              <textarea
                id="ticket-description"
                placeholder="Descreva o problema ou solicitação..."
                value={ticketForm.description}
                onChange={(e) => setTicketForm({ ...ticketForm, description: e.target.value })}
                className="w-full min-h-[100px] px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-vertical"
                required
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <label htmlFor="ticket-priority" className="text-sm font-medium">
                  Prioridade
                </label>
                <select
                  id="ticket-priority"
                  value={ticketForm.priority}
                  onChange={(e) => setTicketForm({ ...ticketForm, priority: e.target.value as any })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                >
                  <option value="low">Baixa</option>
                  <option value="medium">Média</option>
                  <option value="high">Alta</option>
                  <option value="urgent">Urgente</option>
                </select>
              </div>

              <div className="space-y-2">
                <label htmlFor="ticket-category" className="text-sm font-medium">
                  Categoria
                </label>
                <Input
                  id="ticket-category"
                  placeholder="Ex: Suporte Técnico"
                  value={ticketForm.category}
                  onChange={(e) => setTicketForm({ ...ticketForm, category: e.target.value })}
                />
              </div>
            </div>

            <div className="flex justify-end gap-3 pt-4">
              <Button
                type="button"
                variant="outline"
                onClick={() => setIsTicketDialogOpen(false)}
              >
                Cancelar
              </Button>
              <Button
                type="submit"
                className="bg-blue-600 hover:bg-blue-700"
              >
                <Ticket className="w-4 h-4 mr-2" />
                Criar Ticket
              </Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>

      {/* Users Management Modal */}
      {usersModalConversationId && (
        <ConversationUsersModal
          open={isUsersModalOpen}
          onOpenChange={setIsUsersModalOpen}
          conversationId={usersModalConversationId}
          session={usersModalSession}
        />
      )}
    </div>
  );
}