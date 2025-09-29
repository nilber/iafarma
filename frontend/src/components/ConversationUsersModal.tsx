import { useState, useEffect } from "react";
import { Users, Plus, Trash2, Loader2, AlertCircle } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { toast } from "sonner";
import { apiClient } from "@/lib/api/client";
import { User } from "@/lib/api/types";

interface ConversationUsersModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  conversationId: string;
  session?: string; // Optional session parameter
}

export function ConversationUsersModal({ open, onOpenChange, conversationId, session = "" }: ConversationUsersModalProps) {
  const [assignedUsers, setAssignedUsers] = useState<User[]>([]);
  const [availableUsers, setAvailableUsers] = useState<User[]>([]);
  const [selectedUserId, setSelectedUserId] = useState<string>("");
  const [isLoading, setIsLoading] = useState(false);
  const [isAssigning, setIsAssigning] = useState(false);
  const [isUnassigning, setIsUnassigning] = useState<string | null>(null);
  const [proxyEnabled, setProxyEnabled] = useState(false);

  useEffect(() => {
    if (open) {
      loadData();
    }
  }, [open, conversationId]);

  const loadData = async () => {
    setIsLoading(true);
    try {
      // Load assigned users
      const assignedResponse = await apiClient.getConversationUsers(conversationId);
      setAssignedUsers(assignedResponse.users || []);

      // Load available users (all users)
      const usersResponse = await apiClient.getUsers();
      setAvailableUsers(usersResponse.users || []);

      // Check if proxy is enabled
      try {
        const proxyResponse = await apiClient.getTenantSetting('enable_whatsapp_group_proxy');
        setProxyEnabled(proxyResponse.setting?.setting_value === 'true');
      } catch (error) {
        setProxyEnabled(false);
      }
    } catch (error) {
      console.error("Error loading conversation users data:", error);
      toast.error("Erro ao carregar dados");
    } finally {
      setIsLoading(false);
    }
  };

    const handleAssignUser = async () => {
    if (!selectedUserId) {
      toast.error("Selecione um usuário para atribuir");
      return;
    }

    setIsAssigning(true);
    try {
      await apiClient.assignUserToConversation(conversationId, selectedUserId, session);
      toast.success("Usuário atribuído com sucesso");
      setSelectedUserId("");
      loadData();
    } catch (error) {
      console.error("Erro ao atribuir usuário:", error);
      toast.error("Erro ao atribuir usuário");
    } finally {
      setIsAssigning(false);
    }
  };

    const handleUnassignUser = async (userId: string) => {
    try {
      await apiClient.unassignUserFromConversation(conversationId, userId, session);
      toast.success("Usuário removido com sucesso");
      loadData();
    } catch (error) {
      console.error("Erro ao remover usuário:", error);
      toast.error("Erro ao remover usuário");
    }
  };

  const getAvailableUsersToAssign = () => {
    const assignedUserIds = assignedUsers.map(u => u.id);
    return availableUsers.filter(u => !assignedUserIds.includes(u.id));
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Users className="w-5 h-5" />
            Gerenciar Usuários da Conversa
          </DialogTitle>
          <DialogDescription>
            {proxyEnabled ? (
              "Gerencie os usuários que participam desta conversa através de grupos WhatsApp"
            ) : (
              "Proxy de grupos WhatsApp não está ativo. Ative nas configurações do sistema para usar esta funcionalidade."
            )}
          </DialogDescription>
        </DialogHeader>

        {!proxyEnabled && (
          <div className="bg-yellow-50 p-4 rounded-lg border border-yellow-200">
            <div className="flex items-start space-x-2">
              <AlertCircle className="w-5 h-5 text-yellow-600 mt-0.5 flex-shrink-0" />
              <div className="text-sm text-yellow-800">
                <p className="font-medium mb-1">Funcionalidade não ativa</p>
                <p>Para usar grupos WhatsApp como proxy, ative a configuração "Conversas por Grupos WhatsApp" nas configurações do sistema.</p>
              </div>
            </div>
          </div>
        )}

        <div className="space-y-6">
          {/* Assigned Users */}
          <div>
            <h3 className="text-lg font-medium mb-3">Usuários Atribuídos ({assignedUsers.length})</h3>
            {isLoading ? (
              <div className="flex items-center justify-center py-8">
                <Loader2 className="w-6 h-6 animate-spin" />
              </div>
            ) : assignedUsers.length === 0 ? (
              <p className="text-muted-foreground text-center py-8">Nenhum usuário atribuído</p>
            ) : (
              <div className="space-y-2">
                {assignedUsers.map((user) => (
                  <div key={user.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                    <div className="flex items-center gap-3">
                      <Avatar className="w-8 h-8">
                        <AvatarFallback>{user.name.charAt(0).toUpperCase()}</AvatarFallback>
                      </Avatar>
                      <div>
                        <p className="font-medium">{user.name}</p>
                        <p className="text-sm text-muted-foreground">{user.email}</p>
                      </div>
                      <Badge variant="secondary">{user.role}</Badge>
                    </div>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleUnassignUser(user.id)}
                      disabled={isUnassigning === user.id || !proxyEnabled}
                    >
                      {isUnassigning === user.id ? (
                        <Loader2 className="w-4 h-4 animate-spin" />
                      ) : (
                        <Trash2 className="w-4 h-4" />
                      )}
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Add User */}
          {proxyEnabled && (
            <div>
              <h3 className="text-lg font-medium mb-3">Atribuir Usuário</h3>
              <div className="flex gap-2">
                <div className="flex-1">
                  <Select value={selectedUserId} onValueChange={setSelectedUserId}>
                    <SelectTrigger>
                      <SelectValue placeholder="Selecione um usuário" />
                    </SelectTrigger>
                    <SelectContent>
                      {getAvailableUsersToAssign().map((user) => (
                        <SelectItem key={user.id} value={user.id}>
                          <div className="flex items-center gap-2">
                            <span>{user.name}</span>
                            <Badge variant="outline" className="ml-auto">{user.role}</Badge>
                          </div>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <Button
                  onClick={handleAssignUser}
                  disabled={!selectedUserId || isAssigning || !session}
                >
                  {isAssigning ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <Plus className="w-4 h-4" />
                  )}
                </Button>
              </div>
              {!session && (
                <p className="text-sm text-red-600 mt-1">
                  Nenhuma sessão WhatsApp ativa encontrada
                </p>
              )}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
