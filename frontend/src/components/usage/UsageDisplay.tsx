import React from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { MessageSquare, Package, Users, Zap, Wifi, AlertTriangle, CheckCircle } from 'lucide-react';

interface UsageDisplayProps {
  usage?: {
    plan: {
      name: string;
      max_conversations: number;
      max_products: number;
      max_channels: number;
      max_messages_per_month: number;
      max_credits_per_month: number;
    };
    conversations_count: number;
    products_count: number;
    channels_count: number;
    messages_used: number;
    credits_used: number;
    billing_cycle_start: string;
    billing_cycle_end: string;
  };
  loading?: boolean;
  error?: string;
}

const UsageDisplay: React.FC<UsageDisplayProps> = ({ usage, loading, error }) => {
  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Uso do Plano</CardTitle>
          <CardDescription>Carregando...</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4 animate-pulse">
            {[1, 2, 3, 4, 5].map((i) => (
              <div key={i} className="space-y-2">
                <div className="h-4 bg-gray-200 rounded w-3/4"></div>
                <div className="h-2 bg-gray-200 rounded"></div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Alert variant="destructive">
        <AlertTriangle className="h-4 w-4" />
        <AlertDescription>{error}</AlertDescription>
      </Alert>
    );
  }

  if (!usage) {
    return (
      <Alert>
        <AlertTriangle className="h-4 w-4" />
        <AlertDescription>Nenhum dado de uso disponível.</AlertDescription>
      </Alert>
    );
  }

  const getUsagePercentage = (current: number, max: number) => {
    return max > 0 ? Math.min(Math.round((current / max) * 100), 100) : 0;
  };

  const getUsageColor = (percentage: number) => {
    if (percentage >= 90) return 'text-red-600';
    if (percentage >= 80) return 'text-yellow-600';
    return 'text-green-600';
  };

  const getProgressColor = (percentage: number) => {
    if (percentage >= 90) return 'bg-red-500';
    if (percentage >= 80) return 'bg-yellow-500';
    return 'bg-green-500';
  };

  const resources = [
    {
      icon: Users,
      label: 'Conversas',
      current: usage.conversations_count,
      max: usage.plan.max_conversations,
      description: 'Conversas ativas simultâneas',
    },
    {
      icon: Package,
      label: 'Produtos',
      current: usage.products_count,
      max: usage.plan.max_products,
      description: 'Produtos cadastrados no catálogo',
    },
    {
      icon: Wifi,
      label: 'Canais',
      current: usage.channels_count,
      max: usage.plan.max_channels,
      description: 'Canais de comunicação conectados',
    },
    {
      icon: MessageSquare,
      label: 'Mensagens',
      current: usage.messages_used,
      max: usage.plan.max_messages_per_month,
      description: 'Mensagens enviadas este mês',
      monthly: true,
    },
    {
      icon: Zap,
      label: 'Créditos',
      current: usage.credits_used,
      max: usage.plan.max_credits_per_month,
      description: 'Créditos de IA utilizados este mês',
      monthly: true,
    },
  ];

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('pt-BR', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
    });
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Uso do Plano - {usage.plan.name}</CardTitle>
              <CardDescription>
                Período de cobrança: {formatDate(usage.billing_cycle_start)} até {formatDate(usage.billing_cycle_end)}
              </CardDescription>
            </div>
            <Badge variant="outline" className="bg-primary/10">
              Plano Ativo
            </Badge>
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {resources.map((resource) => {
              const percentage = getUsagePercentage(resource.current, resource.max);
              const isOverLimit = resource.current > resource.max;
              
              return (
                <div key={resource.label} className="space-y-3 p-4 border rounded-lg">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center space-x-2">
                      <resource.icon className="h-4 w-4 text-primary" />
                      <span className="font-medium text-sm">{resource.label}</span>
                      {resource.monthly && (
                        <Badge variant="secondary" className="text-xs">
                          Mensal
                        </Badge>
                      )}
                    </div>
                    {isOverLimit ? (
                      <AlertTriangle className="h-4 w-4 text-red-500" />
                    ) : percentage >= 90 ? (
                      <AlertTriangle className="h-4 w-4 text-yellow-500" />
                    ) : (
                      <CheckCircle className="h-4 w-4 text-green-500" />
                    )}
                  </div>
                  
                  <div className="space-y-2">
                    <div className="flex justify-between text-sm">
                      <span className={getUsageColor(percentage)}>
                        {resource.current.toLocaleString()} / {resource.max.toLocaleString()}
                      </span>
                      <span className={`font-medium ${getUsageColor(percentage)}`}>
                        {percentage}%
                      </span>
                    </div>
                    
                    <Progress 
                      value={Math.min(percentage, 100)} 
                      className="h-2"
                    />
                    
                    {isOverLimit && (
                      <Alert variant="destructive" className="">
                        <AlertTriangle className="h-3 w-3" />
                        <AlertDescription className="text-xs">
                          Limite excedido! Considere fazer upgrade do plano.
                        </AlertDescription>
                      </Alert>
                    )}
                  </div>
                  
                  <p className="text-xs text-gray-500">{resource.description}</p>
                </div>
              );
            })}
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default UsageDisplay;
