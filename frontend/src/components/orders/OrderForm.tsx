import { useState, useEffect } from 'react';
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
import { Combobox } from '@/components/ui/combobox';
import { OrderItem, OrderWithCustomer, Customer, Product } from '@/lib/api/types';
import { toast } from 'sonner';

// Extended OrderItem interface for form use
interface FormOrderItem extends OrderItem {
  product_name?: string;
  unit_price?: number;
  total_price?: number;
}

interface OrderFormProps {
  order?: OrderWithCustomer;
  onSuccess: () => void;
  onCancel: () => void;
}

export function OrderForm({ order, onSuccess, onCancel }: OrderFormProps) {
  const [status, setStatus] = useState(order?.status || 'pending');
  const [paymentStatus, setPaymentStatus] = useState(order?.payment_status || 'pending');
  const [fulfillmentStatus, setFulfillmentStatus] = useState(order?.fulfillment_status || 'pending');
  const [notes, setNotes] = useState(order?.notes || '');
  const [items, setItems] = useState<FormOrderItem[]>(
    order?.items?.map(item => ({
      ...item,
      unit_price: parseFloat(item.price || '0'),
      total_price: parseFloat(item.total || '0'),
    })) || []
  );
  const [customerId, setCustomerId] = useState(order?.customer_id || '');
  const [shippingStreet, setShippingStreet] = useState(order?.shipping_street || '');
  const [shippingNumber, setShippingNumber] = useState(order?.shipping_number || '');
  const [shippingComplement, setShippingComplement] = useState(order?.shipping_complement || '');
  const [shippingNeighborhood, setShippingNeighborhood] = useState(order?.shipping_neighborhood || '');
  const [shippingCity, setShippingCity] = useState(order?.shipping_city || '');
  const [shippingState, setShippingState] = useState(order?.shipping_state || '');
  const [shippingZipcode, setShippingZipcode] = useState(order?.shipping_zipcode || '');
  const queryClient = useQueryClient();

  // Fetch customers for the customer selector
  const { data: customersData } = useQuery({
    queryKey: ['customers'],
    queryFn: () => apiClient.getCustomers({ limit: 100 }),
  });

  // Fetch products for the product selector
  const { data: productsData } = useQuery({
    queryKey: ['products'],
    queryFn: () => apiClient.getProducts({ limit: 100 }),
  });

  const customers = customersData?.data || [];
  const products = productsData?.data || [];

  // Create or update order mutation
  const createOrderMutation = useMutation({
    mutationFn: (orderData: any) => apiClient.createOrder(orderData),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      toast.success('Pedido criado com sucesso!');
      onSuccess();
    },
    onError: (error: any) => {
      toast.error('Erro ao criar pedido: ' + (error.message || 'Erro desconhecido'));
    },
  });

  const updateOrderMutation = useMutation({
    mutationFn: ({ id, ...orderData }: any) => apiClient.updateOrder(id, orderData),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      queryClient.invalidateQueries({ queryKey: ['order', order?.id] });
      toast.success('Pedido atualizado com sucesso!');
      onSuccess();
    },
    onError: (error: any) => {
      toast.error('Erro ao atualizar pedido: ' + (error.message || 'Erro desconhecido'));
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!customerId) {
      toast.error('Selecione um cliente');
      return;
    }

    if (items.length === 0) {
      toast.error('Adicione pelo menos um produto ao pedido');
      return;
    }

    const shippingAddress = [
      shippingStreet,
      shippingNumber,
      shippingComplement,
      shippingNeighborhood,
      shippingCity,
      shippingState,
      shippingZipcode
    ].filter(Boolean).join(', ');

    const orderData = {
      customer_id: customerId,
      status,
      payment_status: paymentStatus,
      fulfillment_status: fulfillmentStatus,
      notes,
      shipping_street: shippingStreet,
      shipping_number: shippingNumber,
      shipping_complement: shippingComplement,
      shipping_neighborhood: shippingNeighborhood,
      shipping_city: shippingCity,
      shipping_state: shippingState,
      shipping_zipcode: shippingZipcode,
      items: items.map(item => ({
        product_id: item.product_id,
        quantity: item.quantity,
        price: (item.unit_price || 0).toString(),
        total: (item.total_price || 0).toString(),
      })),
    };

    if (order) {
      updateOrderMutation.mutate({ id: order.id, ...orderData });
    } else {
      createOrderMutation.mutate(orderData);
    }
  };

  const addItem = () => {
    setItems([...items, {
      id: `temp-${Date.now()}`,
      product_id: '',
      product_name: '',
      quantity: 1,
      price: '0',
      total: '0',
      unit_price: 0,
      total_price: 0,
    }]);
  };

  const removeItem = (index: number) => {
    setItems(items.filter((_, i) => i !== index));
  };

  const updateItem = (index: number, field: keyof FormOrderItem, value: any) => {
    const newItems = [...items];
    newItems[index] = { ...newItems[index], [field]: value };
    
    // Auto-calculate total when quantity or unit_price changes
    if (field === 'quantity' || field === 'unit_price') {
      const unitPrice = field === 'unit_price' ? value : newItems[index].unit_price || 0;
      const quantity = field === 'quantity' ? value : newItems[index].quantity;
      const totalPrice = quantity * unitPrice;
      
      newItems[index].unit_price = unitPrice;
      newItems[index].total_price = totalPrice;
      newItems[index].price = unitPrice.toString();
      newItems[index].total = totalPrice.toString();
    }
    
    // Auto-fill product name when product_id changes
    if (field === 'product_id') {
      const product = products.find(p => p.id === value);
      if (product) {
        const unitPrice = parseFloat(product.price || '0');
        newItems[index].product_name = product.name;
        newItems[index].unit_price = unitPrice;
        newItems[index].price = unitPrice.toString();
        newItems[index].total_price = newItems[index].quantity * unitPrice;
        newItems[index].total = newItems[index].total_price.toString();
      }
    }
    
    setItems(newItems);
  };

  const customerOptions = customers.map((customer: Customer) => ({
    value: customer.id,
    label: `${customer.name} - ${customer.phone}`,
  }));

  const productOptions = products.map((product: Product) => ({
    value: product.id,
    label: `${product.name} - R$ ${product.price}`,
  }));

  const totalAmount = items.reduce((sum, item) => sum + (item.total_price || 0), 0);

  const isLoading = createOrderMutation.isPending || updateOrderMutation.isPending;

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* Customer Selection */}
      <div className="space-y-2">
        <Label htmlFor="customer">Cliente *</Label>
        <Combobox
          options={customerOptions}
          value={customerId}
          onValueChange={setCustomerId}
          placeholder="Selecione um cliente..."
          searchPlaceholder="Buscar cliente..."
          emptyText="Nenhum cliente encontrado"
        />
      </div>

      {/* Order Status */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="space-y-2">
          <Label htmlFor="status">Status do Pedido</Label>
          <Select value={status} onValueChange={setStatus}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="pending">Pendente</SelectItem>
              <SelectItem value="confirmed">Confirmado</SelectItem>
              <SelectItem value="preparing">Preparando</SelectItem>
              <SelectItem value="shipped">Enviado</SelectItem>
              <SelectItem value="delivered">Entregue</SelectItem>
              <SelectItem value="cancelled">Cancelado</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-2">
          <Label htmlFor="payment_status">Status do Pagamento</Label>
          <Select value={paymentStatus} onValueChange={setPaymentStatus}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="pending">Pendente</SelectItem>
              <SelectItem value="paid">Pago</SelectItem>
              <SelectItem value="failed">Falhou</SelectItem>
              <SelectItem value="refunded">Reembolsado</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-2">
          <Label htmlFor="fulfillment_status">Status de Entrega</Label>
          <Select value={fulfillmentStatus} onValueChange={setFulfillmentStatus}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="pending">Pendente</SelectItem>
              <SelectItem value="processing">Processando</SelectItem>
              <SelectItem value="shipped">Enviado</SelectItem>
              <SelectItem value="delivered">Entregue</SelectItem>
              <SelectItem value="cancelled">Cancelado</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* Shipping Address */}
      <Card>
        <CardHeader>
          <CardTitle>Endereço de Entrega</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="shipping_street">Rua</Label>
              <Input
                id="shipping_street"
                value={shippingStreet}
                onChange={(e) => setShippingStreet(e.target.value)}
                placeholder="Nome da rua"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="shipping_number">Número</Label>
              <Input
                id="shipping_number"
                value={shippingNumber}
                onChange={(e) => setShippingNumber(e.target.value)}
                placeholder="Número"
              />
            </div>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="shipping_complement">Complemento</Label>
              <Input
                id="shipping_complement"
                value={shippingComplement}
                onChange={(e) => setShippingComplement(e.target.value)}
                placeholder="Apartamento, bloco, etc."
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="shipping_neighborhood">Bairro</Label>
              <Input
                id="shipping_neighborhood"
                value={shippingNeighborhood}
                onChange={(e) => setShippingNeighborhood(e.target.value)}
                placeholder="Bairro"
              />
            </div>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="space-y-2">
              <Label htmlFor="shipping_city">Cidade</Label>
              <Input
                id="shipping_city"
                value={shippingCity}
                onChange={(e) => setShippingCity(e.target.value)}
                placeholder="Cidade"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="shipping_state">Estado</Label>
              <Input
                id="shipping_state"
                value={shippingState}
                onChange={(e) => setShippingState(e.target.value)}
                placeholder="Estado"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="shipping_zipcode">CEP</Label>
              <Input
                id="shipping_zipcode"
                value={shippingZipcode}
                onChange={(e) => setShippingZipcode(e.target.value)}
                placeholder="CEP"
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Order Items */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Itens do Pedido</CardTitle>
            <Button type="button" onClick={addItem} size="sm">
              <Plus className="w-4 h-4 mr-2" />
              Adicionar Item
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <ScrollArea className="max-h-[400px]">
            <div className="space-y-4">
              {items.map((item, index) => (
                <div key={index} className="grid grid-cols-12 gap-2 items-end p-3 border rounded-lg">
                  <div className="col-span-5">
                    <Label>Produto</Label>
                    <Combobox
                      options={productOptions}
                      value={item.product_id}
                      onValueChange={(value) => updateItem(index, 'product_id', value)}
                      placeholder="Selecione um produto..."
                      searchPlaceholder="Buscar produto..."
                      emptyText="Nenhum produto encontrado"
                    />
                  </div>
                  <div className="col-span-2">
                    <Label>Quantidade</Label>
                    <Input
                      type="number"
                      min="1"
                      value={item.quantity}
                      onChange={(e) => updateItem(index, 'quantity', parseInt(e.target.value) || 1)}
                    />
                  </div>
                  <div className="col-span-2">
                    <Label>Preço Unit.</Label>
                    <Input
                      type="number"
                      step="0.01"
                      min="0"
                      value={item.unit_price || 0}
                      onChange={(e) => updateItem(index, 'unit_price', parseFloat(e.target.value) || 0)}
                    />
                  </div>
                  <div className="col-span-2">
                    <Label>Total</Label>
                    <Input
                      type="text"
                      value={`R$ ${(item.total_price || 0).toFixed(2)}`}
                      readOnly
                      className="bg-muted"
                    />
                  </div>
                  <div className="col-span-1">
                    <Button
                      type="button"
                      variant="outline"
                      size="icon"
                      onClick={() => removeItem(index)}
                      className="text-destructive hover:text-destructive"
                    >
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>
                </div>
              ))}
              
              {items.length === 0 && (
                <div className="text-center py-8 text-muted-foreground">
                  Nenhum item adicionado. Clique em "Adicionar Item" para começar.
                </div>
              )}
            </div>
          </ScrollArea>
          
          {items.length > 0 && (
            <div className="mt-4 pt-4 border-t">
              <div className="flex justify-between items-center text-lg font-semibold">
                <span>Total do Pedido:</span>
                <span>R$ {totalAmount.toFixed(2)}</span>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Notes */}
      <div className="space-y-2">
        <Label htmlFor="notes">Observações</Label>
        <Textarea
          id="notes"
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
          placeholder="Observações adicionais sobre o pedido..."
          rows={3}
        />
      </div>

      {/* Actions */}
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancelar
        </Button>
        <Button type="submit" disabled={isLoading}>
          {isLoading && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
          {order ? 'Atualizar Pedido' : 'Criar Pedido'}
        </Button>
      </div>
    </form>
  );
}
