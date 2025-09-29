import { useMemo, memo, useCallback } from "react";
import { Zap, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { apiClient } from "@/lib/api/client";
import { useAuth } from "@/contexts/AuthContext";
import { useQuery } from "@tanstack/react-query";
// Debug hooks removidos

export function AICreditsDisplay() {
  const { user } = useAuth();
  
  // Use React Query para cache e evitar reloads desnecessários
  const { data: credits, isLoading: loading } = useQuery({
    queryKey: ['ai-credits'],
    queryFn: () => apiClient.getAICredits(),
    staleTime: 1000 * 60 * 5, // 5 minutes cache
    refetchInterval: 1000 * 60 * 5, // Refetch every 5 minutes
    refetchOnWindowFocus: false,
  });

  const handleAddCredits = useCallback(() => {
    // TODO: Implementar modal para adicionar créditos
    console.log('Adicionar créditos');
  }, []);

  // Memoize computed values to prevent unnecessary re-renders
  // IMPORTANTE: Todos os hooks devem ser chamados antes de cualquer early return
  const isLowCredits = useMemo(() => 
    credits ? credits.remaining_credits < 10 : false, 
    [credits?.remaining_credits]
  );
  const isAdmin = useMemo(() => user?.role === 'admin', [user?.role]);

    // Debug hooks removidos para eliminar re-renders

  if (loading) {
    return (
      <div className="flex items-center gap-2 px-3 py-2 bg-background/50 rounded-lg border">
        <Zap className="w-4 h-4 text-yellow-500" />
        <span className="text-sm text-muted-foreground">Carregando...</span>
      </div>
    );
  }

  if (!credits) {
    return null;
  }

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="flex items-center gap-2 px-3 py-2 bg-background/50 rounded-lg border">
            <Zap className="w-4 h-4 text-yellow-500" />
            <div className="flex items-center gap-2">
              <Badge 
                variant={isLowCredits ? "destructive" : "secondary"}
                className="text-xs"
              >
                {credits.remaining_credits} créditos
              </Badge>
              {isAdmin && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleAddCredits}
                  className="h-6 w-6 p-0"
                >
                  <Plus className="w-3 h-3" />
                </Button>
              )}
            </div>
          </div>
        </TooltipTrigger>
        <TooltipContent>
          <div className="text-sm">
            <p><strong>Créditos de IA:</strong> {credits.remaining_credits}</p>
            <p><strong>Total usado:</strong> {credits.used_credits}</p>
            <p className="text-xs text-muted-foreground mt-1">
              Use para gerar descrições de produtos com IA
            </p>
            {isLowCredits && (
              <p className="text-xs text-destructive mt-1">
                ⚠️ Poucos créditos restantes
              </p>
            )}
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

// Memoize the component to prevent unnecessary re-renders
export default memo(AICreditsDisplay);
