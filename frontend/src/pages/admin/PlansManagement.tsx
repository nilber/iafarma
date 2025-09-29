import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Switch } from '@/components/ui/switch';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Trash2, Edit2, Plus, CheckCircle, XCircle, Crown, Star, Users, Copy } from 'lucide-react';
import { apiClient } from '@/lib/api';
import { useAuth } from '@/contexts/AuthContext';

interface Plan {
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
  features: string[];
  is_active: boolean;
  is_default: boolean;
  stripe_url: string;
  created_at: string;
  updated_at: string;
}

interface PlanFormData {
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
  stripe_url: string;
}

const PlansManagement: React.FC = () => {
  const { user } = useAuth();
  const [plans, setPlans] = useState<Plan[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [validationErrors, setValidationErrors] = useState<{[key: string]: string}>({});
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [editingPlan, setEditingPlan] = useState<Plan | null>(null);
  const [formData, setFormData] = useState<PlanFormData>({
    name: '',
    description: '',
    price: 0,
    currency: 'BRL',
    billing_period: 'monthly',
    max_conversations: 50,
    max_products: 100,
    max_channels: 1,
    max_messages_per_month: 1000,
    max_credits_per_month: 500,
    features: '',
    is_active: true,
    is_default: false,
    stripe_url: '',
  });

  // Verificar se é system admin
  if (!user || user.role !== 'system_admin') {
    return (
      <div className="p-6">
        <Alert>
          <XCircle className="h-4 w-4" />
          <AlertDescription>
            Acesso negado. Esta página é apenas para Administradores do Sistema.
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  const fetchPlans = async () => {
    try {
      setLoading(true);
      const response = await apiClient.getPlans();
      
      // Garantir que plans é sempre um array
      let plans = Array.isArray(response) ? response : [];
      
      // Processar features para garantir que seja sempre um array
      plans = plans.map(plan => ({
        ...plan,
        features: typeof plan.features === 'string' 
          ? (plan.features.trim() ? 
              // Tentar JSON parse primeiro, se falhar, tratar como string separada por vírgulas
              (plan.features.startsWith('{') || plan.features.startsWith('[') ? 
                (() => {
                  try {
                    const parsed = JSON.parse(plan.features);
                    return Array.isArray(parsed) ? parsed : Object.values(parsed);
                  } catch {
                    return plan.features.split(',').map(f => f.trim()).filter(f => f);
                  }
                })() :
                plan.features.split(',').map(f => f.trim()).filter(f => f)
              ) : [])
          : Array.isArray(plan.features) 
            ? plan.features 
            : []
      }));
      
      // Ordenar planos por preço crescente
      plans.sort((a, b) => a.price - b.price);
      
      setPlans(plans);
      setError(null);
    } catch (err: any) {
      setError(err.message || 'Erro ao carregar planos');
      setPlans([]); // Definir array vazio em caso de erro
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchPlans();
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setValidationErrors({});
    
    try {
      const planData = {
        ...formData,
        features: formData.features, // Manter como string, não converter para array
      };

      if (editingPlan) {
        await apiClient.updatePlan(editingPlan.id, planData);
      } else {
        await apiClient.createPlan(planData);
      }

      await fetchPlans(); // Aguardar a atualização
      setIsDialogOpen(false);
      setEditingPlan(null);
      resetForm();
    } catch (err: any) {
      console.log('Error caught:', err);
      
      if (err.response?.status === 400 && err.response?.data?.validation_errors) {
        // Erro de validação com detalhes específicos
        setValidationErrors(err.response.data.validation_errors);
        setError(err.response.data.message || 'Dados do plano inválidos');
      } else {
        // Outros tipos de erro
        setError(err.response?.data?.message || err.message || 'Erro ao salvar plano');
      }
    }
  };

  const handleEdit = (plan: Plan) => {
    setEditingPlan(plan);
    setFormData({
      name: plan.name,
      description: plan.description,
      price: plan.price,
      currency: plan.currency,
      billing_period: plan.billing_period,
      max_conversations: plan.max_conversations,
      max_products: plan.max_products,
      max_channels: plan.max_channels,
      max_messages_per_month: plan.max_messages_per_month,
      max_credits_per_month: plan.max_credits_per_month,
      features: plan.features.join(', '),
      is_active: plan.is_active,
      is_default: plan.is_default,
      stripe_url: plan.stripe_url || '',
    });
    setIsDialogOpen(true);
  };

  const handleDelete = async (planId: string) => {
    if (!confirm('Tem certeza que deseja excluir este plano?')) return;

    try {
      await apiClient.deletePlan(planId);
      fetchPlans();
    } catch (err: any) {
      setError(err.message || 'Erro ao excluir plano');
    }
  };

  const resetForm = () => {
    setFormData({
      name: '',
      description: '',
      price: 0,
      currency: 'BRL',
      billing_period: 'monthly',
      max_conversations: 50,
      max_products: 100,
      max_channels: 1,
      max_messages_per_month: 1000,
      max_credits_per_month: 500,
      features: '',
      is_active: true,
      is_default: false,
      stripe_url: '',
    });
    setError(null);
    setValidationErrors({});
  };

  const formatPrice = (price: number, currency: string) => {
    return new Intl.NumberFormat('pt-BR', {
      style: 'currency',
      currency: currency || 'BRL',
    }).format(price);
  };

  const getPlanIcon = (planName: string) => {
    const name = planName.toLowerCase();
    if (name.includes('grátis') || name.includes('free')) {
      return <Star className="h-5 w-5 text-yellow-500" />;
    }
    if (name.includes('premium') || name.includes('pro')) {
      return <Crown className="h-5 w-5 text-purple-500" />;
    }
    return <Users className="h-5 w-5 text-blue-500" />;
  };

  const copyToClipboard = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      // You could add a toast notification here if you have one
      console.log('ID copied to clipboard');
    } catch (err) {
      console.error('Failed to copy: ', err);
    }
  };

  if (loading) {
    return (
      <div className="p-6">
        <div className="animate-pulse space-y-4">
          <div className="h-8 bg-gray-200 rounded w-1/4"></div>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {[1, 2, 3].map((i) => (
              <div key={i} className="h-64 bg-gray-200 rounded"></div>
            ))}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold">Gerenciamento de Planos</h1>
          <p className="text-gray-600">Gerencie os planos de assinatura do sistema</p>
        </div>
        <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
          <DialogTrigger asChild>
            <Button onClick={() => { resetForm(); setEditingPlan(null); }}>
              <Plus className="h-4 w-4 mr-2" />
              Novo Plano
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-2xl">
            <DialogHeader>
              <DialogTitle>
                {editingPlan ? 'Editar Plano' : 'Criar Novo Plano'}
              </DialogTitle>
              <DialogDescription>
                Configure os limites e recursos do plano de assinatura.
              </DialogDescription>
            </DialogHeader>
            
            {error && (
              <Alert variant="destructive" className="mb-4">
                <XCircle className="h-4 w-4" />
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
            
            <form onSubmit={handleSubmit}>
              <div className="grid grid-cols-2 gap-4 py-4">
                <div className="space-y-2">
                  <Label htmlFor="name">Nome do Plano</Label>
                  <Input
                    id="name"
                    value={formData.name}
                    onChange={(e) => setFormData({...formData, name: e.target.value})}
                    required
                    className={validationErrors.name ? "border-red-500" : ""}
                  />
                  {validationErrors.name && (
                    <p className="text-sm text-red-500">{validationErrors.name}</p>
                  )}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="price">Preço</Label>
                  <Input
                    id="price"
                    type="number"
                    step="0.01"
                    value={formData.price}
                    onChange={(e) => setFormData({...formData, price: parseFloat(e.target.value)})}
                    required
                    className={validationErrors.price ? "border-red-500" : ""}
                  />
                  {validationErrors.price && (
                    <p className="text-sm text-red-500">{validationErrors.price}</p>
                  )}
                </div>
                <div className="col-span-2 space-y-2">
                  <Label htmlFor="description">Descrição</Label>
                  <Textarea
                    id="description"
                    value={formData.description}
                    onChange={(e) => setFormData({...formData, description: e.target.value})}
                  />
                </div>
                <div className="col-span-2 space-y-2">
                  <Label htmlFor="stripe_url">URL do Stripe</Label>
                  <Input
                    id="stripe_url"
                    type="url"
                    value={formData.stripe_url}
                    onChange={(e) => setFormData({...formData, stripe_url: e.target.value})}
                    placeholder="https://buy.stripe.com/..."
                  />
                  <p className="text-xs text-gray-500">URL do link de pagamento do Stripe</p>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="max_conversations">Max Conversas</Label>
                  <Input
                    id="max_conversations"
                    type="number"
                    value={formData.max_conversations}
                    onChange={(e) => setFormData({...formData, max_conversations: parseInt(e.target.value)})}
                    required
                    className={validationErrors.max_conversations ? "border-red-500" : ""}
                  />
                  {validationErrors.max_conversations && (
                    <p className="text-sm text-red-500">{validationErrors.max_conversations}</p>
                  )}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="max_products">Max Produtos</Label>
                  <Input
                    id="max_products"
                    type="number"
                    value={formData.max_products}
                    onChange={(e) => setFormData({...formData, max_products: parseInt(e.target.value)})}
                    required
                    className={validationErrors.max_products ? "border-red-500" : ""}
                  />
                  {validationErrors.max_products && (
                    <p className="text-sm text-red-500">{validationErrors.max_products}</p>
                  )}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="max_channels">Max Canais</Label>
                  <Input
                    id="max_channels"
                    type="number"
                    value={formData.max_channels}
                    onChange={(e) => setFormData({...formData, max_channels: parseInt(e.target.value)})}
                    required
                    className={validationErrors.max_channels ? "border-red-500" : ""}
                  />
                  {validationErrors.max_channels && (
                    <p className="text-sm text-red-500">{validationErrors.max_channels}</p>
                  )}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="max_messages_per_month">Mensagens/Mês</Label>
                  <Input
                    id="max_messages_per_month"
                    type="number"
                    value={formData.max_messages_per_month}
                    onChange={(e) => setFormData({...formData, max_messages_per_month: parseInt(e.target.value)})}
                    required
                    className={validationErrors.max_messages_per_month ? "border-red-500" : ""}
                  />
                  {validationErrors.max_messages_per_month && (
                    <p className="text-sm text-red-500">{validationErrors.max_messages_per_month}</p>
                  )}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="max_credits_per_month">Créditos/Mês</Label>
                  <Input
                    id="max_credits_per_month"
                    type="number"
                    value={formData.max_credits_per_month}
                    onChange={(e) => setFormData({...formData, max_credits_per_month: parseInt(e.target.value)})}
                    required
                    className={validationErrors.max_credits_per_month ? "border-red-500" : ""}
                  />
                  {validationErrors.max_credits_per_month && (
                    <p className="text-sm text-red-500">{validationErrors.max_credits_per_month}</p>
                  )}
                </div>
                <div className="col-span-2 space-y-2">
                  <Label htmlFor="features">Recursos (separados por vírgula)</Label>
                  <Textarea
                    id="features"
                    value={formData.features}
                    onChange={(e) => setFormData({...formData, features: e.target.value})}
                    placeholder="Suporte 24/7, API Access, Analytics..."
                  />
                </div>
                <div className="flex items-center space-x-2">
                  <Switch
                    id="is_active"
                    checked={formData.is_active}
                    onCheckedChange={(checked) => setFormData({...formData, is_active: checked})}
                  />
                  <Label htmlFor="is_active">Plano Ativo</Label>
                </div>
                <div className="flex items-center space-x-2">
                  <Switch
                    id="is_default"
                    checked={formData.is_default}
                    onCheckedChange={(checked) => setFormData({...formData, is_default: checked})}
                  />
                  <Label htmlFor="is_default">Plano Padrão</Label>
                </div>
              </div>
              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setIsDialogOpen(false)}>
                  Cancelar
                </Button>
                <Button type="submit">
                  {editingPlan ? 'Atualizar' : 'Criar'} Plano
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      {error && (
        <Alert variant="destructive">
          <XCircle className="h-4 w-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {Array.isArray(plans) && plans.length > 0 ? (
          plans.map((plan) => (
            <Card key={plan.id} className={`relative ${plan.is_default ? 'border-primary border-2' : ''}`}>
              {plan.is_default && (
                <div className="absolute -top-2 left-1/2 transform -translate-x-1/2">
                  <Badge variant="default" className="bg-primary">
                    Plano Padrão
                  </Badge>
                </div>
              )}
            <CardHeader>
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-2">
                  {getPlanIcon(plan.name)}
                  <CardTitle className="text-xl">{plan.name}</CardTitle>
                </div>
                <div className="flex items-center space-x-2">
                  {plan.is_active ? (
                    <Badge variant="outline" className="text-green-600 border-green-600">
                      <CheckCircle className="h-3 w-3 mr-1" />
                      Ativo
                    </Badge>
                  ) : (
                    <Badge variant="outline" className="text-red-600 border-red-600">
                      <XCircle className="h-3 w-3 mr-1" />
                      Inativo
                    </Badge>
                  )}
                </div>
              </div>
              <CardDescription>{plan.description}</CardDescription>
              <div className="text-xs text-gray-500 mb-2 flex items-center gap-2">
                <span className="font-mono bg-gray-100 px-2 py-1 rounded">ID: {plan.id.slice(0, 8)}...</span>
                <button
                  onClick={() => copyToClipboard(plan.id)}
                  className="p-1 hover:bg-gray-100 rounded transition-colors"
                  title="Copiar ID completo"
                >
                  <Copy className="h-3 w-3" />
                </button>
              </div>
              <div className="text-3xl font-bold text-primary">
                {formatPrice(plan.price, plan.currency)}
                <span className="text-sm font-normal text-gray-500">/{plan.billing_period === 'monthly' ? 'mês' : 'ano'}</span>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span>Conversas:</span>
                  <span className="font-medium">{plan.max_conversations}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span>Produtos:</span>
                  <span className="font-medium">{plan.max_products}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span>Canais:</span>
                  <span className="font-medium">{plan.max_channels}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span>Mensagens/mês:</span>
                  <span className="font-medium">{plan.max_messages_per_month.toLocaleString()}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span>Créditos/mês:</span>
                  <span className="font-medium">{plan.max_credits_per_month.toLocaleString()}</span>
                </div>
              </div>

              {plan.features.length > 0 && (
                <>
                  <Separator />
                  <div className="space-y-1">
                    <p className="text-sm font-medium">Recursos inclusos:</p>
                    <ul className="text-xs space-y-1">
                      {plan.features.map((feature, index) => (
                        <li key={index} className="flex items-center">
                          <CheckCircle className="h-3 w-3 text-green-500 mr-1" />
                          {feature}
                        </li>
                      ))}
                    </ul>
                  </div>
                </>
              )}

              <Separator />
              <div className="flex justify-between">
                <Button variant="outline" size="sm" onClick={() => handleEdit(plan)}>
                  <Edit2 className="h-4 w-4 mr-1" />
                  Editar
                </Button>
                <Button variant="destructive" size="sm" onClick={() => handleDelete(plan.id)}>
                  <Trash2 className="h-4 w-4 mr-1" />
                  Excluir
                </Button>
              </div>
            </CardContent>
          </Card>
          ))
        ) : (
          <div className="col-span-full">
            <Card>
              <CardContent className="text-center py-12">
                <p className="text-gray-500 mb-4">Nenhum plano cadastrado</p>
                <Button onClick={() => setIsDialogOpen(true)}>
                  <Plus className="h-4 w-4 mr-2" />
                  Criar Primeiro Plano
                </Button>
              </CardContent>
            </Card>
          </div>
        )}
      </div>

      {/* Formulário de criação/edição de plano */}
    </div>
  );
};

export default PlansManagement;
