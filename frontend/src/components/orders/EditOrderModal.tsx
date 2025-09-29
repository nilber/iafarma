import { useState, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Loader2, Plus, Trash2, MapPin } from 'lucide-react';
import { useMutation, useQueryClient, useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import { useToast } from '@/hooks/use-toast';
import { Combobox } from '@/components/ui/combobox';
import { OrderItem } from '@/lib/api/types';

interface EditOrderModalProps {
  orderId: string | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function EditOrderModal({ orderId, open, onOpenChange }: EditOrderModalProps) {
  const [status, setStatus] = useState('');
  const [paymentStatus, setPaymentStatus] = useState('');
  const [fulfillmentStatus, setFulfillmentStatus] = useState('');
  const [notes, setNotes] = useState('');
  const [items, setItems] = useState<OrderItem[]>([]);
  const { toast } = useToast();
  const queryClient = useQueryClient();

  // Fetch order details
  const { data: orderData, isLoading } = useQuery({
    queryKey: ['order', orderId],
    queryFn: () => apiClient.getOrder(orderId!),
    enabled: !!orderId && open,
  });

  // Fetch products for item selection
  const { data: productsData } = useQuery({
    queryKey: ['products'],
    queryFn: () => apiClient.getProducts(),
    enabled: open,
  });

  const order = orderData;
  const products = productsData?.data || [];

  // Update order mutation
  const updateOrderMutation = useMutation({
    mutationFn: (orderData: any) => apiClient.updateOrder(orderId!, orderData),
    onSuccess: () => {
      toast({
        title: "Pedido atualizado",
        description: "O pedido foi atualizado com sucesso.",
      });
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      queryClient.invalidateQueries({ queryKey: ['order', orderId] });
      queryClient.invalidateQueries({ queryKey: ['order-stats'] });
      onOpenChange(false);
    },
    onError: (error: any) => {
      toast({
        title: "Erro ao atualizar pedido",
        description: error.message || "Ocorreu um erro ao atualizar o pedido.",
        variant: "destructive",
      });
    },
  });

  // Load order data when it changes
  useEffect(() => {
    if (order) {
      setStatus(order.status || '');
      setPaymentStatus(order.payment_status || '');
      setFulfillmentStatus(order.fulfillment_status || '');
      setNotes(order.notes || '');
      setItems(order.items || []);
    }
  }, [order]);

  // Item management functions
  const addItem = () => {
    const newItem: OrderItem = {
      product_id: '',
      quantity: 1,
      price: '0',
      total: '0',
    };
    setItems([...items, newItem]);
  };

  const removeItem = (index: number) => {
    setItems(items.filter((_, i) => i !== index));
  };

  const updateItem = (index: number, field: keyof OrderItem, value: any) => {
    const updatedItems = [...items];
    
    // When product is selected, auto-fill price information
    if (field === 'product_id') {
      const product = products.find(p => p.id === value);
      if (product) {
        updatedItems[index] = { 
          ...updatedItems[index], 
          [field]: value,
          price: product.sale_price || product.price, // Use sale price if available
        };
        
        // Calculate total with new price
        const price = parseFloat(product.sale_price || product.price || '0');
        const quantity = updatedItems[index].quantity;
        updatedItems[index].total = (price * quantity).toFixed(2);
      } else {
        updatedItems[index] = { ...updatedItems[index], [field]: value };
      }
    } else {
      updatedItems[index] = { ...updatedItems[index], [field]: value };
      
      // Auto-calculate total when price or quantity changes
      if (field === 'price' || field === 'quantity') {
        const price = parseFloat(field === 'price' ? value : updatedItems[index].price || '0');
        const quantity = parseInt(field === 'quantity' ? value : updatedItems[index].quantity.toString());
        updatedItems[index].total = (price * quantity).toFixed(2);
      }
    }
    
    setItems(updatedItems);
  };

  const calculateOrderTotal = () => {
    return items.reduce((total, item) => total + parseFloat(item.total || '0'), 0).toFixed(2);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    
    const updates: any = {
      status,
      payment_status: paymentStatus,
      fulfillment_status: fulfillmentStatus,
      notes,
      items,
      total_amount: calculateOrderTotal(),
      subtotal: calculateOrderTotal(), // For simplicity, assuming no tax/shipping
    };

    // Add timestamps based on status changes
    if (fulfillmentStatus === 'shipped' && order?.fulfillment_status !== 'shipped') {
      updates.shipped_at = new Date().toISOString();
    }
    if (fulfillmentStatus === 'delivered' && order?.fulfillment_status !== 'delivered') {
      updates.delivered_at = new Date().toISOString();
    }

    updateOrderMutation.mutate(updates);
  };

  if (!orderId) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Editar Pedido</DialogTitle>
          <DialogDescription>
            Atualize as informações do pedido #{order?.order_number || orderId}
          </DialogDescription>
        </DialogHeader>

        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="w-6 h-6 animate-spin" />
          </div>
        ) : (
          <ScrollArea className="max-h-[70vh]">
            <form onSubmit={handleSubmit} className="space-y-6">
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                {/* Left Column - Order Status */}
                <div className="space-y-4">
                  <Card>
                    <CardHeader>
                      <CardTitle className="text-lg">Status do Pedido</CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-4">
                      {/* Status */}
                      <div className="space-y-2">
                        <Label>Status</Label>
                        <Select value={status} onValueChange={setStatus}>
                          <SelectTrigger>
                            <SelectValue placeholder="Selecione o status" />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="pending">Pendente</SelectItem>
                            <SelectItem value="processing">Processando</SelectItem>
                            <SelectItem value="completed">Concluído</SelectItem>
                            <SelectItem value="cancelled">Cancelado</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>

                      {/* Payment Status */}
                      <div className="space-y-2">
                        <Label>Status do Pagamento</Label>
                        <Select value={paymentStatus} onValueChange={setPaymentStatus}>
                          <SelectTrigger>
                            <SelectValue placeholder="Selecione o status do pagamento" />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="pending">Pendente</SelectItem>
                            <SelectItem value="paid">Pago</SelectItem>
                            <SelectItem value="failed">Falhou</SelectItem>
                            <SelectItem value="refunded">Reembolsado</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>

                      {/* Fulfillment Status */}
                      <div className="space-y-2">
                        <Label>Status de Entrega</Label>
                        <Select value={fulfillmentStatus} onValueChange={setFulfillmentStatus}>
                          <SelectTrigger>
                            <SelectValue placeholder="Selecione o status de entrega" />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="pending">Pendente</SelectItem>
                            <SelectItem value="unfulfilled">Não Enviado</SelectItem>
                            <SelectItem value="shipped">Enviado</SelectItem>
                            <SelectItem value="delivered">Entregue</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>

                      {/* Notes */}
                      <div className="space-y-2">
                        <Label>Observações</Label>
                        <Textarea
                          value={notes}
                          onChange={(e) => setNotes(e.target.value)}
                          placeholder="Observações sobre o pedido..."
                          rows={3}
                        />
                      </div>
                    </CardContent>
                  </Card>

                  {/* Shipping Address */}
                  {(order?.shipping_street || order?.shipping_city) && (
                    <Card>
                      <CardHeader>
                        <CardTitle className="text-lg flex items-center gap-2">
                          <MapPin className="w-5 h-5" />
                          Endereço de Entrega
                        </CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="space-y-2 text-sm">
                          {order.shipping_name && <p><strong>Nome:</strong> {order.shipping_name}</p>}
                          {order.shipping_street && (
                            <p>
                              <strong>Endereço:</strong> {order.shipping_street}
                              {order.shipping_number && `, ${order.shipping_number}`}
                              {order.shipping_complement && `, ${order.shipping_complement}`}
                            </p>
                          )}
                          {order.shipping_neighborhood && <p><strong>Bairro:</strong> {order.shipping_neighborhood}</p>}
                          {order.shipping_city && order.shipping_state && (
                            <p><strong>Cidade:</strong> {order.shipping_city} - {order.shipping_state}</p>
                          )}
                          {order.shipping_zipcode && <p><strong>CEP:</strong> {order.shipping_zipcode}</p>}
                        </div>
                      </CardContent>
                    </Card>
                  )}
                </div>

                {/* Right Column - Order Items */}
                <div className="space-y-4">
                  <Card>
                    <CardHeader>
                      <CardTitle className="text-lg flex items-center justify-between">
                        Itens do Pedido
                        <Button type="button" onClick={addItem} size="sm" variant="outline">
                          <Plus className="w-4 h-4 mr-2" />
                          Adicionar Item
                        </Button>
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-4 max-h-96 overflow-y-auto">
                        {items.map((item, index) => (
                          <div key={index} className="border p-4 rounded-lg space-y-3">
                            <div className="flex justify-between items-start">
                              <h4 className="font-medium">Item {index + 1}</h4>
                              <Button
                                type="button"
                                onClick={() => removeItem(index)}
                                size="sm"
                                variant="outline"
                                className="text-red-600"
                              >
                                <Trash2 className="w-4 h-4" />
                              </Button>
                            </div>

                            {/* Product Selection */}
                            <div className="space-y-2">
                              <Label>Produto</Label>
                              <Combobox
                                value={item.product_id}
                                onValueChange={(value) => updateItem(index, 'product_id', value)}
                                placeholder="Selecione um produto"
                                options={products.map((product) => ({
                                  value: product.id,
                                  label: product.name,
                                }))}
                              />
                            </div>

                            {/* Quantity */}
                            <div className="space-y-2">
                              <Label>Quantidade</Label>
                              <Input
                                type="number"
                                min="1"
                                value={item.quantity}
                                onChange={(e) => updateItem(index, 'quantity', parseInt(e.target.value) || 1)}
                              />
                            </div>

                            {/* Price */}
                            <div className="space-y-2">
                              <Label>Preço Unitário (R$)</Label>
                              {(() => {
                                const product = products.find(p => p.id === item.product_id);
                                const isPromotion = product?.sale_price && product.sale_price !== product.price;
                                
                                return (
                                  <div className="space-y-1">
                                    <Input
                                      type="number"
                                      step="0.01"
                                      min="0"
                                      value={parseFloat(item.price || '0')}
                                      readOnly
                                      className="bg-gray-50"
                                    />
                                    {isPromotion && (
                                      <p className="text-xs text-orange-600 font-medium">
                                        🏷️ Preço promocional aplicado
                                      </p>
                                    )}
                                  </div>
                                );
                              })()}
                            </div>

                            {/* Total */}
                            <div className="space-y-2">
                              <Label>Total (R$)</Label>
                              <Input
                                type="text"
                                value={item.total}
                                readOnly
                                className="bg-gray-50"
                              />
                            </div>
                          </div>
                        ))}
                        
                        {items.length === 0 && (
                          <div className="text-center py-8 text-gray-500">
                            Nenhum item adicionado. Clique em "Adicionar Item" para começar.
                          </div>
                        )}
                      </div>

                      {/* Order Total */}
                      {items.length > 0 && (
                        <div className="mt-4 pt-4 border-t">
                          <div className="flex justify-between items-center font-semibold">
                            <span>Total do Pedido:</span>
                            <span>R$ {calculateOrderTotal()}</span>
                          </div>
                        </div>
                      )}
                    </CardContent>
                  </Card>
                </div>
              </div>

              {/* Action Buttons */}
              <div className="flex justify-end space-x-2 pt-4 border-t">
                <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                  Cancelar
                </Button>
                <Button type="submit" disabled={updateOrderMutation.isPending}>
                  {updateOrderMutation.isPending ? (
                    <>
                      <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                      Salvando...
                    </>
                  ) : (
                    'Salvar Alterações'
                  )}
                </Button>
              </div>
            </form>
          </ScrollArea>
        )}
      </DialogContent>
    </Dialog>
  );
}
