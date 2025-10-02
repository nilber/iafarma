import React, { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { 
  Dialog, 
  DialogContent, 
  DialogDescription, 
  DialogHeader, 
  DialogTitle 
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { 
  CheckCircle2, 
  Circle, 
  Store, 
  MessageSquare, 
  Package,
  ArrowRight,
  X,
  Sparkles,
  Link,
  CreditCard
} from 'lucide-react';
import { apiClient } from '@/lib/api/client';
import { toast } from 'sonner';

// Types
interface OnboardingItem {
  id: string;
  title: string;
  description: string;
  is_completed: boolean;
  action_url: string;
  priority: number;
  completed_at?: string;
}

interface OnboardingStatus {
  is_completed: boolean;
  completion_rate: number;
  items: OnboardingItem[];
  tenant_created_at: string;
}

interface OnboardingModalProps {
  isOpen: boolean;
  onClose: () => void;
  onDismiss?: () => void;
  status?: OnboardingStatus | null;
  loading?: boolean;
}

// Icon mapping for different onboarding items
const getItemIcon = (itemId: string) => {
  switch (itemId) {
    case 'store_config':
      return Store;
    case 'whatsapp_channel':
      return MessageSquare;
    case 'products':
      return Package;
    case 'payment_methods':
      return CreditCard;
    case 'channel_connection':
      return Link;
    default:
      return Circle;
  }
};

// Color mapping for completion status
const getItemColor = (isCompleted: boolean) => {
  return isCompleted ? 'text-green-600' : 'text-gray-400';
};

export function OnboardingModal({ 
  isOpen, 
  onClose, 
  onDismiss, 
  status: propStatus, 
  loading: propLoading = false 
}: OnboardingModalProps) {
  const [dismissing, setDismissing] = useState(false);
  const navigate = useNavigate();

  // Use prop status if provided, otherwise maintain internal state for backward compatibility
  const status = propStatus;
  const loading = propLoading;

  // Handle action button click
  const handleActionClick = (actionUrl: string) => {
    onClose();
    navigate(actionUrl);
  };

  // Handle dismiss onboarding
  const handleDismiss = async () => {
    if (!onDismiss) return;
    
    try {
      setDismissing(true);
      await apiClient.dismissOnboarding();
      toast.success('Onboarding dispensado temporariamente');
      onDismiss();
      onClose();
    } catch (error: any) {
      console.error('Error dismissing onboarding:', error);
      toast.error('Erro ao dispensar onboarding');
    } finally {
      setDismissing(false);
    }
  };

  if (!status && !loading) return null;

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <div className="flex items-center gap-2">
            <Sparkles className="h-6 w-6 text-primary" />
            <DialogTitle className="text-2xl">Bem-vindo ao IAFarma!</DialogTitle>
          </div>
          <DialogDescription className="text-base">
            Para começar a usar o sistema completamente, você precisa fazer algumas configurações iniciais.
          </DialogDescription>
        </DialogHeader>

        {loading ? (
          <div className="flex items-center justify-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
          </div>
        ) : status ? (
          <div className="space-y-6">
            {/* Progress Overview */}
            <Card>
              <CardHeader className="pb-3">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-lg">Progresso Geral</CardTitle>
                  <Badge 
                    variant={status.is_completed ? "default" : "secondary"}
                    className="text-sm"
                  >
                    {Math.round(status.completion_rate)}% concluído
                  </Badge>
                </div>
              </CardHeader>
              <CardContent>
                <Progress 
                  value={status.completion_rate} 
                  className="h-3"
                />
                <p className="text-sm text-muted-foreground mt-2">
                  {status.items.filter(item => item.is_completed).length} de {status.items.length} itens concluídos
                </p>
              </CardContent>
            </Card>

            {/* Onboarding Items */}
            <div className="space-y-3">
              <h3 className="text-lg font-semibold">Configurações Necessárias</h3>
              
              {status.items
                .sort((a, b) => a.priority - b.priority)
                .map((item) => {
                  const ItemIcon = getItemIcon(item.id);
                  const iconColor = getItemColor(item.is_completed);
                  
                  return (
                    <Card key={item.id} className={`transition-all duration-200 ${
                      item.is_completed 
                        ? 'border-green-200 bg-green-50/50' 
                        : 'border-gray-200 hover:border-primary/50'
                    }`}>
                      <CardContent className="p-4">
                        <div className="flex items-start justify-between gap-4">
                          <div className="flex items-start gap-3 flex-1">
                            <div className={`mt-1 ${iconColor}`}>
                              {item.is_completed ? (
                                <CheckCircle2 className="h-5 w-5" />
                              ) : (
                                <ItemIcon className="h-5 w-5" />
                              )}
                            </div>
                            
                            <div className="flex-1">
                              <h4 className={`font-medium ${
                                item.is_completed ? 'text-green-800' : 'text-gray-900'
                              }`}>
                                {item.title}
                              </h4>
                              <p className="text-sm text-muted-foreground mt-1">
                                {item.description}
                              </p>
                              
                              {item.completed_at && (
                                <p className="text-xs text-green-600 mt-2">
                                  ✓ Concluído em {new Date(item.completed_at).toLocaleDateString('pt-BR')}
                                </p>
                              )}
                            </div>
                          </div>
                          
                          {!item.is_completed && (
                            <Button
                              size="sm"
                              onClick={() => handleActionClick(item.action_url)}
                              className="flex items-center gap-2 shrink-0"
                            >
                              Configurar
                              <ArrowRight className="h-4 w-4" />
                            </Button>
                          )}
                        </div>
                      </CardContent>
                    </Card>
                  );
                })}
            </div>

            {/* Completion Message */}
            {status.is_completed && (
              <Card className="border-green-200 bg-green-50">
                <CardContent className="p-4">
                  <div className="flex items-center gap-3">
                    <CheckCircle2 className="h-6 w-6 text-green-600" />
                    <div>
                      <h4 className="font-medium text-green-800">
                        Parabéns! Configuração completa
                      </h4>
                      <p className="text-sm text-green-700">
                        Seu sistema está pronto para uso. Você já pode começar a receber pedidos!
                      </p>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )}

            {/* Actions */}
            <div className="flex justify-between items-center pt-4 border-t">
              {onDismiss && !status.is_completed && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleDismiss}
                  disabled={dismissing}
                  className="text-muted-foreground hover:text-foreground"
                >
                  {dismissing ? 'Dispensando...' : 'Dispensar por agora'}
                </Button>
              )}
              
              <div className="flex gap-2 ml-auto">
                <Button
                  variant="outline"
                  onClick={onClose}
                >
                  {status.is_completed ? 'Fechar' : 'Fechar'}
                </Button>
                
                {!status.is_completed && (
                  <Button
                    onClick={() => {
                      const nextIncomplete = status.items
                        .sort((a, b) => a.priority - b.priority)
                        .find(item => !item.is_completed);
                      
                      if (nextIncomplete) {
                        handleActionClick(nextIncomplete.action_url);
                      }
                    }}
                  >
                    Começar configuração
                  </Button>
                )}
              </div>
            </div>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

// Hook to manage onboarding state
export function useOnboarding() {
  const [shouldShowOnboarding, setShouldShowOnboarding] = useState(false);
  const [onboardingStatus, setOnboardingStatus] = useState<OnboardingStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const isCheckingRef = useRef(false);
  const hasCheckedRef = useRef(false);

  const checkOnboardingStatus = async () => {
    // Prevent multiple calls - only check once per session
    if (hasCheckedRef.current || isCheckingRef.current) return;
    
    try {
      isCheckingRef.current = true;
      hasCheckedRef.current = true;
      setLoading(true);
      const status = await apiClient.getOnboardingStatus();
      
      setOnboardingStatus(status);
      
      // Show onboarding if not completed and tenant is relatively new (< 7 days)
      const tenantAge = Date.now() - new Date(status.tenant_created_at).getTime();
      const isNewTenant = tenantAge < 7 * 24 * 60 * 60 * 1000; // 7 days
      
      setShouldShowOnboarding(!status.is_completed && isNewTenant);
    } catch (error: any) {
      console.error('Error checking onboarding status:', error);
      setShouldShowOnboarding(false);
    } finally {
      setLoading(false);
      isCheckingRef.current = false;
    }
  };

  const dismissOnboarding = () => {
    setShouldShowOnboarding(false);
  };

  return {
    shouldShowOnboarding,
    onboardingStatus,
    loading,
    checkOnboardingStatus,
    dismissOnboarding,
    setShouldShowOnboarding
  };
}