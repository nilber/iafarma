import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { ArrowLeft, ShoppingCart, Save, Loader2 } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { useOrder } from "@/lib/api/hooks";
import { OrderForm } from "@/components/orders/OrderForm";
import { OrderWithCustomer } from "@/lib/api/types";
import { toast } from "sonner";

export default function OrderEditDetails() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isNew = id === "new";

  // Only fetch order if we have an ID and it's not "new"
  const shouldFetchOrder = !isNew && !!id;
  const { data: order, isLoading, error } = useOrder(shouldFetchOrder ? id! : "skip");

  const handleSuccess = () => {
    toast.success(isNew ? "Pedido criado com sucesso!" : "Pedido atualizado com sucesso!");
    navigate("/sales/orders");
  };

  const handleCancel = () => {
    navigate("/sales/orders");
  };

  if (!isNew && isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="sm" onClick={() => navigate("/sales/orders")}>
            <ArrowLeft className="w-4 h-4 mr-2" />
            Voltar para Pedidos
          </Button>
        </div>
        
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-center min-h-[400px]">
              <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (!isNew && error) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="sm" onClick={() => navigate("/sales/orders")}>
            <ArrowLeft className="w-4 h-4 mr-2" />
            Voltar para Pedidos
          </Button>
        </div>
        
        <Card>
          <CardContent className="p-6">
            <div className="text-center py-8">
              <ShoppingCart className="w-12 h-12 text-muted-foreground mx-auto mb-4" />
              <h3 className="text-lg font-semibold mb-2">Pedido não encontrado</h3>
              <p className="text-muted-foreground mb-4">
                O pedido solicitado não foi encontrado ou você não tem permissão para visualizá-lo.
              </p>
              <Button onClick={() => navigate("/sales/orders")}>
                Voltar para Pedidos
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="sm" onClick={() => navigate("/sales/orders")}>
            <ArrowLeft className="w-4 h-4 mr-2" />
            Voltar para Pedidos
          </Button>
          
          <div>
            <h1 className="text-2xl font-bold tracking-tight">
              {isNew ? "Novo Pedido" : "Editar Pedido"}
            </h1>
            <p className="text-muted-foreground">
              {isNew 
                ? "Crie um novo pedido de venda"
                : `Atualize as informações do pedido${order?.id ? ` #${order.id.slice(-8)}` : ""}`
              }
            </p>
          </div>
        </div>
      </div>

      {/* Order Form */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <ShoppingCart className="w-5 h-5" />
            Informações do Pedido
          </CardTitle>
          <CardDescription>
            Preencha as informações do pedido abaixo
          </CardDescription>
        </CardHeader>
        <CardContent>
          <OrderForm
            order={isNew ? undefined : order}
            onSuccess={handleSuccess}
            onCancel={handleCancel}
          />
        </CardContent>
      </Card>
    </div>
  );
}
