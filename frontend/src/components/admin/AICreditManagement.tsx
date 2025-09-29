import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Plus, Zap, History } from "lucide-react";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { apiClient } from "@/lib/api/client";
import { AICredits, AICreditTransaction, AddCreditsRequest } from "@/lib/api/types";
import { toast } from "sonner";
import { useAuth } from "@/contexts/AuthContext";

export function AICreditManagement() {
  const { user } = useAuth();
  const [credits, setCredits] = useState<AICredits | null>(null);
  const [transactions, setTransactions] = useState<AICreditTransaction[]>([]);
  const [loading, setLoading] = useState(false);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [addCreditsForm, setAddCreditsForm] = useState({
    amount: "",
    description: ""
  });

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);
      const [creditsData, transactionsResponse] = await Promise.all([
        apiClient.getAICredits(),
        apiClient.getAICreditTransactions()
      ]);
      setCredits(creditsData);
      setTransactions(transactionsResponse.transactions);
    } catch (error) {
      console.error("Erro ao carregar dados:", error);
      toast.error("Erro ao carregar dados dos créditos");
    } finally {
      setLoading(false);
    }
  };

  const handleAddCredits = async () => {
    if (!addCreditsForm.amount || !addCreditsForm.description) {
      toast.error("Preencha todos os campos");
      return;
    }

    try {
      const request: AddCreditsRequest = {
        tenant_id: user?.tenant_id || "",
        amount: parseInt(addCreditsForm.amount),
        description: addCreditsForm.description
      };

      await apiClient.addAICredits(request);
      toast.success("Créditos adicionados com sucesso!");
      
      setAddCreditsForm({ amount: "", description: "" });
      setDialogOpen(false);
      loadData();
    } catch (error) {
      toast.error("Erro ao adicionar créditos");
      console.error("Erro:", error);
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString('pt-BR');
  };

  const getTransactionBadgeVariant = (type: string) => {
    switch (type) {
      case 'add': return 'default';
      case 'use': return 'destructive';
      case 'refund': return 'secondary';
      default: return 'outline';
    }
  };

  const getTransactionLabel = (type: string) => {
    switch (type) {
      case 'add': return 'Adição';
      case 'use': return 'Uso';
      case 'refund': return 'Reembolso';
      default: return type;
    }
  };

  if (loading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center h-48">
          <div className="text-center">
            <Zap className="w-8 h-8 animate-pulse mx-auto mb-2 text-yellow-500" />
            <p className="text-muted-foreground">Carregando...</p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Credits Overview */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Créditos Disponíveis</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold flex items-center gap-2">
              <Zap className="w-6 h-6 text-yellow-500" />
              {credits?.remaining_credits || 0}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Total de Créditos</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-muted-foreground">
              {credits?.total_credits || 0}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Créditos Usados</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-destructive">
              {credits?.used_credits || 0}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Add Credits */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Plus className="w-5 h-5" />
            Adicionar Créditos
          </CardTitle>
          <CardDescription>
            Adicione créditos para permitir o uso da IA na geração de produtos
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
            <DialogTrigger asChild>
              <Button className="w-full">
                <Plus className="w-4 h-4 mr-2" />
                Adicionar Créditos
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Adicionar Créditos de IA</DialogTitle>
                <DialogDescription>
                  Adicione créditos para permitir a geração de produtos com IA
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="amount">Quantidade de Créditos</Label>
                  <Input
                    id="amount"
                    type="number"
                    min="1"
                    value={addCreditsForm.amount}
                    onChange={(e) => setAddCreditsForm(prev => ({ ...prev, amount: e.target.value }))}
                    placeholder="Ex: 100"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="description">Descrição</Label>
                  <Textarea
                    id="description"
                    value={addCreditsForm.description}
                    onChange={(e) => setAddCreditsForm(prev => ({ ...prev, description: e.target.value }))}
                    placeholder="Ex: Recarga mensal de créditos"
                    rows={3}
                  />
                </div>
                <Button onClick={handleAddCredits} className="w-full">
                  Adicionar Créditos
                </Button>
              </div>
            </DialogContent>
          </Dialog>
        </CardContent>
      </Card>

      {/* Transactions History */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <History className="w-5 h-5" />
            Histórico de Transações
          </CardTitle>
          <CardDescription>
            Histórico de todas as movimentações de créditos de IA
          </CardDescription>
        </CardHeader>
        <CardContent>
          {transactions.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              Nenhuma transação encontrada
            </div>
          ) : (
            <div className="space-y-4">
              {transactions.map((transaction) => (
                <div key={transaction.id} className="flex items-center justify-between p-4 border rounded-lg">
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <Badge variant={getTransactionBadgeVariant(transaction.type)}>
                        {getTransactionLabel(transaction.type)}
                      </Badge>
                      <span className="font-medium">
                        {transaction.type === 'add' ? '+' : '-'}{transaction.amount} créditos
                      </span>
                    </div>
                    <p className="text-sm text-muted-foreground">
                      {transaction.description}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {formatDate(transaction.created_at)}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
