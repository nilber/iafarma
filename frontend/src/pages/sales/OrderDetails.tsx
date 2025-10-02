import { useState, useEffect, useCallback, useMemo } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { ArrowLeft, User, Package, CreditCard, Truck, Calendar, Phone, Mail, MapPin, Edit, Plus, Minus, Printer, X, Search, FileText, Send } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useOrder, useProduct, useProducts, useUpdateOrder, useAddOrderItem, useUpdateOrderItem, useRemoveOrderItem, useAddressesByCustomer, useActivePaymentMethods, useProductCharacteristics } from "@/lib/api/hooks";
import { ProductAutocomplete } from "@/components/ProductAutocomplete";
import { format } from "date-fns";
import { ptBR } from "date-fns/locale";
import { OrderItem, Order, PaymentMethod } from "@/lib/api/types";
import { apiClient } from "@/lib/api/client";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "@/components/ui/use-toast";

// Component to display individual order item with edit capabilities
function OrderItemDisplay({ 
  item, 
  canEdit, 
  onUpdateQuantity, 
  onRemoveItem 
}: { 
  item: OrderItem; 
  canEdit: boolean;
  onUpdateQuantity: (itemId: string, newQuantity: number) => void;
  onRemoveItem: (itemId: string) => void;
}) {
  const { data: product } = useProduct(item.product_id);
  const [quantity, setQuantity] = useState(item.quantity);

  const handleQuantityChange = (newQty: number) => {
    if (newQty < 1) return;
    setQuantity(newQty);
    onUpdateQuantity(item.id, newQty);
  };

  return (
    <div className="flex items-center justify-between p-4 border rounded-lg bg-card">
      <div className="flex items-center gap-4 flex-1">
        <div className="w-12 h-12 bg-muted rounded-lg flex items-center justify-center">
          <Package className="w-6 h-6 text-muted-foreground" />
        </div>
        <div className="flex-1">
          <h4 className="font-medium">
            {item.product_name || product?.name || 'Produto não encontrado'}
          </h4>
          <p className="text-sm text-muted-foreground">
            Código: {item.product_id.substring(0, 8)}
          </p>
          {item.attributes && item.attributes.length > 0 && (
            <div className="text-sm text-muted-foreground mt-1">
              <span className="font-medium">Opções selecionadas:</span>
              <div className="flex flex-wrap gap-1 mt-1">
                {item.attributes.map((attr, index) => (
                  <Badge key={index} variant="outline" className="text-xs">
                    {attr.attribute_name}: {attr.option_name}
                    {attr.option_price && parseFloat(attr.option_price) > 0 && (
                      <span className="ml-1">
                        (+R$ {parseFloat(attr.option_price).toFixed(2).replace('.', ',')})
                      </span>
                    )}
                  </Badge>
                ))}
              </div>
            </div>
          )}
          <p className="text-sm text-muted-foreground">
            Preço unitário: R$ {parseFloat(item.price).toFixed(2).replace('.', ',')}
          </p>
        </div>
      </div>
      
      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2">
          {canEdit && (
            <Button 
              size="sm" 
              variant="outline" 
              className="no-print"
              onClick={() => handleQuantityChange(quantity - 1)}
              disabled={quantity <= 1}
            >
              <Minus className="w-4 h-4" />
            </Button>
          )}
          
          <span className="font-medium w-12 text-center">
            {quantity}
          </span>
          
          {canEdit && (
            <Button 
              size="sm" 
              variant="outline"
              className="no-print"
              onClick={() => handleQuantityChange(quantity + 1)}
            >
              <Plus className="w-4 h-4" />
            </Button>
          )}
        </div>
        
        <div className="text-right min-w-[80px]">
          <p className="font-medium">
            R$ {(parseFloat(item.price) * quantity).toFixed(2).replace('.', ',')}
          </p>
        </div>
        
        {canEdit && (
          <Button 
            size="sm" 
            variant="outline"
            className="no-print text-destructive hover:text-destructive"
            onClick={() => onRemoveItem(item.id)}
          >
            <X className="w-4 h-4" />
          </Button>
        )}
      </div>
    </div>
  );
}

export default function OrderDetails() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: order, isLoading, error, refetch } = useOrder(id!);
  const { data: productsData } = useProducts();
  const queryClient = useQueryClient();
  
  // Order mutations
  const updateOrderMutation = useUpdateOrder();
  const addItemMutation = useAddOrderItem();
  const updateItemMutation = useUpdateOrderItem();
  const removeItemMutation = useRemoveOrderItem();

  // Estados para modais
  const [customerModalOpen, setCustomerModalOpen] = useState(false);
  const [statusModalOpen, setStatusModalOpen] = useState(false);
  const [addressModalOpen, setAddressModalOpen] = useState(false);
  const [addItemModalOpen, setAddItemModalOpen] = useState(false);
  const [emailModalOpen, setEmailModalOpen] = useState(false);

  // Estados para edição
  const [discountAmount, setDiscountAmount] = useState('0');
  const [newStatus, setNewStatus] = useState('');
  const [newPaymentStatus, setNewPaymentStatus] = useState('');
  const [newPaymentMethodId, setNewPaymentMethodId] = useState('');
  const [newFulfillmentStatus, setNewFulfillmentStatus] = useState('');

  // Estados para email
  const [emailRecipient, setEmailRecipient] = useState('');
  const [emailSubject, setEmailSubject] = useState('');
  const [emailMessage, setEmailMessage] = useState('');
  const [emailSending, setEmailSending] = useState(false);

  // Estado para endereço de entrega
  const [shippingAddress, setShippingAddress] = useState({
    shipping_name: '',
    shipping_street: '',
    shipping_number: '',
    shipping_complement: '',
    shipping_neighborhood: '',
    shipping_city: '',
    shipping_state: '',
    shipping_zipcode: '',
  });

  // Estado para adicionar item
  const [newItem, setNewItem] = useState({
    product_id: '',
    quantity: 1,
    price: '0',
    attributes: [] as Array<{
      attribute_id: string;
      option_id: string;
      attribute_name: string;
      option_name: string;
      option_price: string;
    }>
  });

  // Estado para busca de produtos
  const [productSearchValue, setProductSearchValue] = useState('');

  // Estados para endereços do cliente
  const [selectedAddressId, setSelectedAddressId] = useState('');
  const [orderNotes, setOrderNotes] = useState('');

  // Buscar endereços do cliente (apenas se customer_id existir)
  const { data: customerAddresses } = useAddressesByCustomer(order?.customer_id || '', {
    enabled: !!order?.customer_id
  });
  
  // Buscar características do produto selecionado
  const { data: productCharacteristics } = useProductCharacteristics(newItem.product_id || '');
  
  // Buscar formas de pagamento ativas
  const { data: paymentMethods } = useActivePaymentMethods();

  // Atualizar estados quando o pedido carregar
  useEffect(() => {
    if (order) {
      setDiscountAmount(order.discount_amount || '0');
      setNewStatus(order.status || '');
      setNewPaymentStatus(order.payment_status || '');
      setNewPaymentMethodId(order.payment_method_id || 'none');
      setNewFulfillmentStatus(order.fulfillment_status || '');
      
      // Configurar email padrão
      setEmailRecipient(order.customer_email || '');
      setEmailSubject(`Detalhes do Pedido #${order.order_number}`);
      setEmailMessage(`Olá ${order.customer_name || 'Cliente'},\n\nSegue em anexo os detalhes do seu pedido #${order.order_number}.\n\nAtenciosamente,\nEquipe de Vendas`);
      
      setShippingAddress({
        shipping_name: order.shipping_name || '',
        shipping_street: order.shipping_street || '',
        shipping_number: order.shipping_number || '',
        shipping_complement: order.shipping_complement || '',
        shipping_neighborhood: order.shipping_neighborhood || '',
        shipping_city: order.shipping_city || '',
        shipping_state: order.shipping_state || '',
        shipping_zipcode: order.shipping_zipcode || '',
      });
    }
  }, [order]);

  const isPending = order?.status === 'pending';
  const products = productsData?.data || [];

  // Função para atualizar quantidade do item
  const handleUpdateQuantity = async (itemId: string, newQuantity: number) => {
    if (!order) return;
    
    try {
      await updateItemMutation.mutateAsync({
        orderId: order.id,
        itemId: itemId,
        item: { quantity: newQuantity }
      });

      toast({ 
        title: "Quantidade atualizada", 
        description: "A quantidade do item foi atualizada com sucesso." 
      });
    } catch (error) {
      toast({ 
        title: "Erro", 
        description: "Não foi possível atualizar a quantidade.", 
        variant: "destructive" 
      });
    }
  };

  // Função para remover item
  const handleRemoveItem = async (itemId: string) => {
    if (!order) return;
    
    try {
      await removeItemMutation.mutateAsync({
        orderId: order.id,
        itemId: itemId
      });

      toast({ 
        title: "Item removido", 
        description: "O item foi removido do pedido." 
      });
    } catch (error) {
      toast({ 
        title: "Erro", 
        description: "Não foi possível remover o item.", 
        variant: "destructive" 
      });
    }
  };

  // Função para adicionar item
  const handleAddItem = async () => {
    if (!order) return;
    
    try {
      if (!newItem.product_id || newItem.quantity <= 0) {
        toast({ 
          title: "Erro", 
          description: "Por favor, selecione um produto e quantidade válida.", 
          variant: "destructive" 
        });
        return;
      }

      await addItemMutation.mutateAsync({
        orderId: order.id,
        item: {
          product_id: newItem.product_id,
          quantity: newItem.quantity,
          price: newItem.price,
          attributes: newItem.attributes.length > 0 ? newItem.attributes : undefined
        }
      });

      setAddItemModalOpen(false);
      setNewItem({ product_id: '', quantity: 1, price: '0', attributes: [] });
      setProductSearchValue('');
      
      toast({ 
        title: "Item adicionado", 
        description: "O item foi adicionado ao pedido." 
      });
    } catch (error) {
      toast({ 
        title: "Erro", 
        description: "Não foi possível adicionar o item.", 
        variant: "destructive" 
      });
    }
  };

  // Função para quando um produto é selecionado no autocomplete
  const handleProductSelect = useCallback((product: any) => {
    setNewItem(prev => ({
      ...prev,
      product_id: product.id,
      price: product.price,
      attributes: [] // Resetar atributos quando trocar de produto
    }));
  }, []);

  // Função para gerenciar seleção de atributos
  const handleAttributeSelection = useCallback((attributeId: string, optionId: string, attributeName: string, optionName: string, optionPrice: string) => {
    setNewItem(prev => {
      const existingIndex = prev.attributes.findIndex(attr => attr.attribute_id === attributeId);
      const newAttributes = [...prev.attributes];
      
      if (existingIndex >= 0) {
        // Atualizar seleção existente
        newAttributes[existingIndex] = {
          attribute_id: attributeId,
          option_id: optionId,
          attribute_name: attributeName,
          option_name: optionName,
          option_price: optionPrice
        };
      } else {
        // Adicionar nova seleção
        newAttributes.push({
          attribute_id: attributeId,
          option_id: optionId,
          attribute_name: attributeName,
          option_name: optionName,
          option_price: optionPrice
        });
      }
      
      return { ...prev, attributes: newAttributes };
    });
  }, []);

  // Função para calcular o preço total do item incluindo atributos
  const calculateItemTotal = useCallback(() => {
    const basePrice = parseFloat(newItem.price || '0');
    const attributesPrice = newItem.attributes.reduce((total, attr) => {
      return total + parseFloat(attr.option_price || '0');
    }, 0);
    return (basePrice + attributesPrice) * newItem.quantity;
  }, [newItem.price, newItem.attributes, newItem.quantity]);

  // Função para atualizar status
  const handleUpdateStatus = async () => {
    if (!order) return;
    
    try {
      await updateOrderMutation.mutateAsync({ 
        id: order.id, 
        order: { 
          status: newStatus,
          payment_status: newPaymentStatus,
          payment_method_id: newPaymentMethodId === 'none' ? null : newPaymentMethodId,
          fulfillment_status: newFulfillmentStatus,
          discount_amount: discountAmount 
        } 
      });
      setStatusModalOpen(false);
      toast({ title: "Status atualizado", description: "O status do pedido foi atualizado com sucesso." });
    } catch (error) {
      toast({ title: "Erro", description: "Não foi possível atualizar o status.", variant: "destructive" });
    }
  };

  // Função para enviar email
  const handleSendEmail = async () => {
    if (!emailRecipient.trim()) {
      toast({ title: "Erro", description: "Por favor, informe um email válido.", variant: "destructive" });
      return;
    }

    setEmailSending(true);
    try {
      const response = await apiClient.sendOrderEmail({
        order_id: order?.id || '',
        recipient: emailRecipient,
        subject: emailSubject,
        message: emailMessage
      });

      if (!response.success) {
        throw new Error(response.message || 'Falha ao enviar email');
      }

      setEmailModalOpen(false);
      toast({ title: "Email enviado", description: "O email foi enviado com sucesso." });
    } catch (error) {
      toast({ title: "Erro", description: "Não foi possível enviar o email.", variant: "destructive" });
    } finally {
      setEmailSending(false);
    }
  };

  // Função para gerar e abrir PDF em nova aba
  const handleGeneratePDF = () => {
    // Criar uma nova janela com o conteúdo otimizado para PDF
    const printWindow = window.open('', '_blank');
    if (printWindow) {
      printWindow.document.write(`
        <!DOCTYPE html>
        <html>
        <head>
          <title>Pedido #${order?.order_number}</title>
          <style>
            * { margin: 0; padding: 0; box-sizing: border-box; }
            body { font-family: Arial, sans-serif; font-size: 12px; line-height: 1.4; color: #333; }
            .container { max-width: 800px; margin: 20px auto; padding: 20px; }
            .header { text-align: center; border-bottom: 2px solid #000; padding-bottom: 15px; margin-bottom: 20px; }
            .header h1 { font-size: 24px; margin-bottom: 5px; }
            .header p { font-size: 14px; color: #666; }
            .section { margin-bottom: 20px; }
            .section-title { font-size: 16px; font-weight: bold; border-bottom: 1px solid #ccc; padding-bottom: 5px; margin-bottom: 10px; }
            .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; }
            .info-item { margin-bottom: 8px; }
            .info-item strong { display: inline-block; width: 120px; }
            .items-table { width: 100%; border-collapse: collapse; margin-top: 10px; }
            .items-table th, .items-table td { border: 1px solid #ddd; padding: 8px; text-align: left; }
            .items-table th { background-color: #f5f5f5; font-weight: bold; }
            .total-section { margin-top: 20px; text-align: right; }
            .total-item { margin-bottom: 5px; }
            .total-final { font-size: 16px; font-weight: bold; border-top: 2px solid #000; padding-top: 10px; }
            @media print { body { margin: 0; } .container { margin: 0; padding: 15px; } }
          </style>
        </head>
        <body>
          <div class="container">
            <div class="header">
              <h1>DETALHES DO PEDIDO</h1>
              <p>Pedido #${order?.order_number}</p>
              <p>Data: ${order?.created_at ? format(new Date(order.created_at), "dd/MM/yyyy 'às' HH:mm", { locale: ptBR }) : 'N/A'}</p>
            </div>

            <div class="grid">
              <div>
                <div class="section">
                  <div class="section-title">Informações do Cliente</div>
                  <div class="info-item"><strong>Nome:</strong> ${order?.customer?.name || order?.customer_name || 'N/A'}</div>
                  <div class="info-item"><strong>Email:</strong> ${order?.customer?.email || order?.customer_email || 'N/A'}</div>
                  <div class="info-item"><strong>Telefone:</strong> ${order?.customer?.phone || order?.customer_phone || 'N/A'}</div>
                  <div class="info-item"><strong>Documento:</strong> ${order?.customer?.document || order?.customer_document || 'N/A'}</div>
                </div>

                <div class="section">
                  <div class="section-title">Endereço de Entrega</div>
                  <div class="info-item"><strong>Nome:</strong> ${order?.shipping_name || order?.customer?.name || 'N/A'}</div>
                  <div class="info-item"><strong>Endereço:</strong> ${(order?.shipping_street || '') + ' ' + (order?.shipping_number || '')}</div>
                  <div class="info-item"><strong>Complemento:</strong> ${order?.shipping_complement || 'N/A'}</div>
                  <div class="info-item"><strong>Bairro:</strong> ${order?.shipping_neighborhood || 'N/A'}</div>
                  <div class="info-item"><strong>Cidade:</strong> ${order?.shipping_city || 'N/A'} - ${order?.shipping_state || 'N/A'}</div>
                  <div class="info-item"><strong>CEP:</strong> ${order?.shipping_zipcode || 'N/A'}</div>
                </div>
              </div>

              <div>
                <div class="section">
                  <div class="section-title">Status do Pedido</div>
                  <div class="info-item"><strong>Status:</strong> ${getStatusText(order?.status)}</div>
                  <div class="info-item"><strong>Pagamento:</strong> ${getPaymentStatusText(order?.payment_status)}</div>
                  <div class="info-item"><strong>Entrega:</strong> ${getFulfillmentStatusText(order?.fulfillment_status)}</div>
                </div>

                <div class="section">
                  <div class="section-title">Resumo Financeiro</div>
                  <div class="info-item"><strong>Subtotal:</strong> R$ ${parseFloat(order?.subtotal || '0').toFixed(2)}</div>
                  <div class="info-item"><strong>Desconto:</strong> R$ ${parseFloat(order?.discount_amount || '0').toFixed(2)}</div>
                  <div class="info-item"><strong>Frete:</strong> R$ ${parseFloat(order?.shipping_amount || '0').toFixed(2)}</div>
                  <div class="info-item"><strong>Taxa:</strong> R$ ${parseFloat(order?.tax_amount || '0').toFixed(2)}</div>
                  <div class="total-final"><strong>Total:</strong> R$ ${parseFloat(order?.total_amount || '0').toFixed(2)}</div>
                </div>
                
                ${order?.observations || order?.change_for ? `
                <div class="section">
                  <div class="section-title">Observações do Pedido</div>
                  ${order?.observations ? `<div class="info-item"><strong>Observações:</strong> ${order.observations}</div>` : ''}
                  ${order?.change_for ? `<div class="info-item"><strong>Troco para:</strong> R$ ${order.change_for}</div>` : ''}
                </div>
                ` : ''}
              </div>
            </div>

            <div class="section">
              <div class="section-title">Itens do Pedido</div>
              <table class="items-table">
                <thead>
                  <tr>
                    <th>Produto</th>
                    <th>Quantidade</th>
                    <th>Preço Unit.</th>
                    <th>Total</th>
                  </tr>
                </thead>
                <tbody>
                  ${order?.items?.map(item => {
                    // Use the stored product name from order item, fallback to searching in products list, then generic name
                    const productName = item.product_name || 
                                      productsData?.data?.find(p => p.id === item.product_id)?.name || 
                                      'Produto não encontrado';
                    return `
                    <tr>
                      <td>${productName}</td>
                      <td>${item.quantity}</td>
                      <td>R$ ${parseFloat(item.price || '0').toFixed(2)}</td>
                      <td>R$ ${(parseFloat(item.price || '0') * item.quantity).toFixed(2)}</td>
                    </tr>
                  `}).join('') || ''}
                </tbody>
              </table>
            </div>
          </div>
        </body>
        </html>
      `);
      printWindow.document.close();
      printWindow.focus();
      // Adicionar um delay antes de imprimir para garantir que o conteúdo carregue
      setTimeout(() => {
        printWindow.print();
      }, 500);
    }
  };

  // Função para atualizar endereço
  const handleUpdateAddress = async () => {
    try {
      let addressToUpdate;
      
      // Se um endereço foi selecionado dos endereços do cliente
      if (selectedAddressId && customerAddresses?.addresses) {
        const selectedAddress = customerAddresses.addresses.find(addr => addr.id === selectedAddressId);
        if (selectedAddress) {
          addressToUpdate = {
            shipping_name: selectedAddress.name || order.customer?.name || '',
            shipping_street: selectedAddress.street,
            shipping_number: selectedAddress.number,
            shipping_complement: selectedAddress.complement || '',
            shipping_neighborhood: selectedAddress.neighborhood,
            shipping_city: selectedAddress.city,
            shipping_state: selectedAddress.state,
            shipping_zipcode: selectedAddress.zip_code,
            // Adiciona as observações de entrega
            delivery_notes: orderNotes
          };
        }
      } else {
        // Usa o endereço preenchido manualmente
        addressToUpdate = {
          ...shippingAddress,
          delivery_notes: orderNotes
        };
      }

      if (!addressToUpdate) {
        toast({
          title: "Erro",
          description: "Selecione um endereço ou preencha os dados manualmente",
          variant: "destructive",
        });
        return;
      }

      await updateOrderMutation.mutateAsync({ 
        id: order.id, 
        order: addressToUpdate 
      });
      setAddressModalOpen(false);
      toast({
        title: "Sucesso",
        description: "Endereço de entrega e observações atualizados com sucesso!",
      });
    } catch (error) {
      console.error('Error updating address:', error);
      toast({
        title: "Erro",
        description: "Erro ao atualizar o endereço de entrega",
        variant: "destructive",
      });
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-muted-foreground">Carregando detalhes do pedido...</p>
        </div>
      </div>
    );
  }

  if (error || !order) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <Package className="w-12 h-12 text-muted-foreground mx-auto mb-4" />
          <h3 className="text-lg font-semibold mb-2">Pedido não encontrado</h3>
          <p className="text-muted-foreground mb-4">
            Não foi possível encontrar os detalhes deste pedido.
          </p>
          <Button onClick={() => navigate(-1)}>
            <ArrowLeft className="w-4 h-4 mr-2" />
            Voltar
          </Button>
        </div>
      </div>
    );
  }

  const getStatusColor = (status?: string) => {
    switch (status) {
      case 'completed':
        return 'bg-success text-success-foreground';
      case 'processing':
        return 'bg-primary text-primary-foreground';
      case 'pending':
        return 'bg-warning text-warning-foreground';
      case 'cancelled':
        return 'bg-destructive text-destructive-foreground';
      case 'delivered':
        return 'bg-success text-success-foreground';
      default:
        return 'bg-secondary text-secondary-foreground';
    }
  };

  const getStatusText = (status?: string) => {
    switch (status) {
      case 'completed':
        return 'Concluído';
      case 'processing':
        return 'Processando';
      case 'pending':
        return 'Pendente';
      case 'cancelled':
        return 'Cancelado';
      case 'delivered':
        return 'Entregue';
      default:
        return status || 'Pendente';
    }
  };

  const getPaymentStatusColor = (status?: string) => {
    switch (status) {
      case 'paid':
        return 'bg-success text-success-foreground';
      case 'pending':
        return 'bg-warning text-warning-foreground';
      case 'failed':
        return 'bg-destructive text-destructive-foreground';
      default:
        return 'bg-secondary text-secondary-foreground';
    }
  };

  const getPaymentStatusText = (status?: string) => {
    switch (status) {
      case 'paid':
        return 'Pago';
      case 'pending':
        return 'Pendente';
      case 'failed':
        return 'Falhou';
      default:
        return status || 'Pendente';
    }
  };

  const getFulfillmentStatusText = (status?: string) => {
    switch (status) {
      case 'shipped':
        return 'Enviado';
      case 'delivered':
        return 'Entregue';
      case 'pending':
        return 'Pendente';
      case 'cancelled':
        return 'Cancelado';
      case 'unfulfilled':
        return 'Pendente';
      default:
        return status || 'Pendente';
    }
  };

  return (
    <div className="max-w-6xl mx-auto space-y-6 print-container">
      {/* Print Styles */}
      <style>{`
        @media print {
          .no-print { display: none !important; }
          .print-only { display: block !important; }
          
          /* Otimizações para impressão */
          body { 
            font-size: 10px !important; 
            line-height: 1.2 !important;
            margin: 0 !important;
            padding: 0 !important;
          }
          
          .print-header { 
            text-align: center; 
            margin-bottom: 15px; 
            border-bottom: 1px solid #000; 
            padding-bottom: 8px; 
            font-size: 12px;
          }
          
          .print-container {
            max-width: 100% !important;
            margin: 0 !important;
            padding: 10px !important;
          }
          
          /* Compactar cards */
          .print-card {
            margin-bottom: 10px !important;
            border: 1px solid #ddd !important;
            border-radius: 4px !important;
            padding: 8px !important;
          }
          
          .print-card-header {
            font-size: 11px !important;
            font-weight: bold !important;
            margin-bottom: 5px !important;
            padding-bottom: 3px !important;
            border-bottom: 1px solid #eee !important;
          }
          
          .print-card-content {
            font-size: 9px !important;
            line-height: 1.1 !important;
          }
          
          /* Compactar informações do cliente */
          .print-customer-info {
            display: grid !important;
            grid-template-columns: 1fr 1fr !important;
            gap: 8px !important;
          }
          
          /* Compactar itens do pedido */
          .print-item {
            display: flex !important;
            justify-content: space-between !important;
            padding: 2px 0 !important;
            border-bottom: 1px solid #f0f0f0 !important;
            font-size: 8px !important;
          }
          
          .print-item:last-child {
            border-bottom: none !important;
          }
          
          /* Compactar resumo financeiro */
          .print-summary {
            display: flex !important;
            justify-content: space-between !important;
            font-size: 9px !important;
            margin: 2px 0 !important;
          }
          
          .print-summary.total {
            font-weight: bold !important;
            font-size: 10px !important;
            border-top: 1px solid #000 !important;
            padding-top: 3px !important;
          }
          
          /* Compactar endereço */
          .print-address {
            font-size: 8px !important;
            line-height: 1.1 !important;
          }
          
          /* Layout em duas colunas para impressão */
          .print-layout {
            display: grid !important;
            grid-template-columns: 2fr 1fr !important;
            gap: 15px !important;
          }
          
          /* Quebras de página */
          .page-break {
            page-break-before: always;
          }
          
          /* Títulos menores */
          h1, h2, h3, h4, h5, h6 {
            margin: 5px 0 3px 0 !important;
            line-height: 1.1 !important;
          }
          
          /* Espaçamentos reduzidos */
          .space-y-6 > * + * {
            margin-top: 8px !important;
          }
          
          .space-y-4 > * + * {
            margin-top: 4px !important;
          }
          
          .space-y-2 > * + * {
            margin-top: 2px !important;
          }
        }
        .print-only { display: none; }
      `}</style>

      {/* Print Header - Only visible when printing */}
      <div className="print-only print-header">
        <h1 className="text-2xl font-bold">DETALHES DO PEDIDO</h1>
        <p>Pedido #{order.id}</p>
        <p>Data: {order.created_at ? new Date(order.created_at).toLocaleString('pt-BR') : 'N/A'}</p>
      </div>

      {/* Header */}
      <div className="flex items-center justify-between no-print">
        <div className="flex items-center gap-4">
          <Button variant="outline" onClick={() => navigate(-1)}>
            <ArrowLeft className="w-4 h-4 mr-2" />
            Voltar
          </Button>
          <div>
            <h1 className="text-2xl font-bold">Pedido #{order.order_number}</h1>
            <p className="text-muted-foreground">
              Criado em {order.created_at ? format(new Date(order.created_at), "dd/MM/yyyy 'às' HH:mm", { locale: ptBR }) : 'Data não informada'}
            </p>
          </div>
        </div>
        
        <div className="flex items-center gap-4">
          <Badge className={getStatusColor(order.status)}>
            {getStatusText(order.status)}
          </Badge>
          <div className="flex flex-col gap-1">
            <Badge className={getPaymentStatusColor(order.payment_status)}>
              {getPaymentStatusText(order.payment_status)}
            </Badge>
            {order.payment_method?.name && (
              <span className="text-xs text-muted-foreground">
                {order.payment_method.name}
              </span>
            )}
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 print-layout">
        {/* Main Content */}
        <div className="lg:col-span-2 space-y-6">
          {/* Order Items */}
          <Card className="border-0 shadow-custom-md print-card">
            <CardHeader className="print-card-header">
              <CardTitle className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Package className="w-5 h-5 text-primary" />
                  Itens do Pedido
                </div>
                {isPending && (
                  <Dialog open={addItemModalOpen} onOpenChange={setAddItemModalOpen}>
                    <DialogTrigger asChild>
                      <Button size="sm" variant="outline" className="no-print">
                        <Plus className="w-4 h-4 mr-1" />
                        Adicionar Item
                      </Button>
                    </DialogTrigger>
                    <DialogContent className="max-w-lg max-h-[90vh] overflow-y-auto">
                      <DialogHeader>
                        <DialogTitle>Adicionar Item ao Pedido</DialogTitle>
                      </DialogHeader>
                      <div className="space-y-4">
                        <div>
                          <Label htmlFor="product">Produto</Label>
                          <ProductAutocomplete
                            value={productSearchValue}
                            onChange={setProductSearchValue}
                            onProductSelect={handleProductSelect}
                            placeholder="Digite para buscar produtos..."
                          />
                        </div>
                        <div className="grid grid-cols-2 gap-4">
                          <div>
                            <Label htmlFor="quantity">Quantidade</Label>
                            <Input
                              id="quantity"
                              type="number"
                              min={1}
                              value={newItem.quantity}
                              onChange={(e) => setNewItem(prev => ({ ...prev, quantity: parseInt(e.target.value) || 1 }))}
                            />
                          </div>
                          <div>
                            <Label htmlFor="price">Preço Unitário</Label>
                            <Input
                              id="price"
                              type="number"
                              step="0.01"
                              min="0"
                              value={newItem.price}
                              onChange={(e) => setNewItem(prev => ({ ...prev, price: e.target.value }))}
                              placeholder="0.00"
                            />
                          </div>
                        </div>
                        
                        {/* Seção de Atributos */}
                        {newItem.product_id && productCharacteristics && productCharacteristics.length > 0 && (
                          <div className="space-y-3">
                            <Label className="text-sm font-medium">Atributos do Produto</Label>
                            {productCharacteristics.map((characteristic) => (
                              <div key={characteristic.id} className="space-y-2">
                                <Label className="text-xs text-gray-600">{characteristic.title}</Label>
                                <Select
                                  value={newItem.attributes.find(attr => attr.attribute_id === characteristic.id)?.option_id || ''}
                                  onValueChange={(optionId) => {
                                    const selectedOption = characteristic.items?.find(item => item.id === optionId);
                                    if (selectedOption) {
                                      handleAttributeSelection(
                                        characteristic.id,
                                        optionId,
                                        characteristic.title,
                                        selectedOption.name,
                                        selectedOption.price || '0'
                                      );
                                    }
                                  }}
                                >
                                  <SelectTrigger className="h-8">
                                    <SelectValue placeholder={`Selecionar ${characteristic.title.toLowerCase()}`} />
                                  </SelectTrigger>
                                  <SelectContent>
                                    {characteristic.items?.map((option) => (
                                      <SelectItem key={option.id} value={option.id}>
                                        {option.name}
                                        {option.price && parseFloat(option.price) > 0 && (
                                          <span className="text-green-600 ml-2">+R$ {parseFloat(option.price).toFixed(2)}</span>
                                        )}
                                      </SelectItem>
                                    ))}
                                  </SelectContent>
                                </Select>
                              </div>
                            ))}
                          </div>
                        )}
                        
                        {newItem.product_id && (
                          <div className="p-3 bg-gray-50 rounded-lg space-y-2">
                            <div className="text-sm text-gray-600">Resumo do item:</div>
                            <div className="text-xs space-y-1">
                              <div>Preço base: R$ {parseFloat(newItem.price || '0').toFixed(2)}</div>
                              {newItem.attributes.length > 0 && (
                                <div>
                                  <div>Atributos:</div>
                                  {newItem.attributes.map((attr, index) => (
                                    <div key={index} className="ml-2 text-gray-500">
                                      • {attr.option_name}: +R$ {parseFloat(attr.option_price || '0').toFixed(2)}
                                    </div>
                                  ))}
                                </div>
                              )}
                              <div>Quantidade: {newItem.quantity}</div>
                            </div>
                            <div className="border-t pt-2">
                              <div className="text-lg font-semibold">
                                Total: R$ {calculateItemTotal().toFixed(2)}
                              </div>
                            </div>
                          </div>
                        )}
                        <div className="flex gap-2">
                          <Button 
                            onClick={handleAddItem} 
                            className="flex-1"
                            disabled={!newItem.product_id || newItem.quantity <= 0 || addItemMutation.isPending}
                          >
                            {addItemMutation.isPending ? "Adicionando..." : "Adicionar Item"}
                          </Button>
                          <Button variant="outline" onClick={() => setAddItemModalOpen(false)}>
                            Cancelar
                          </Button>
                        </div>
                      </div>
                    </DialogContent>
                  </Dialog>
                )}
              </CardTitle>
              <CardDescription>
                {order.items?.length || 0} {(order.items?.length || 0) === 1 ? 'item' : 'itens'}
              </CardDescription>
            </CardHeader>
            <CardContent className="print-card-content">
              {order.items && order.items.length > 0 ? (
                <div className="space-y-4">
                  {order.items.map((item, index) => (
                    <div key={item.id || index} className="print-item">
                      <OrderItemDisplay 
                        item={item}
                        canEdit={isPending}
                        onUpdateQuantity={handleUpdateQuantity}
                        onRemoveItem={handleRemoveItem}
                      />
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-8 text-muted-foreground">
                  <Package className="w-12 h-12 mx-auto mb-4 opacity-50" />
                  <p>Nenhum item encontrado neste pedido</p>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Shipping Address */}
          <Card className="border-0 shadow-custom-md print-card">
            <CardHeader className="print-card-header">
              <CardTitle className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <MapPin className="w-5 h-5 text-primary" />
                  Endereço de Entrega
                </div>
                {isPending && (
                  <Dialog open={addressModalOpen} onOpenChange={setAddressModalOpen}>
                    <DialogTrigger asChild>
                      <Button size="sm" variant="outline" className="no-print">
                        <Edit className="w-4 h-4 mr-1" />
                        Editar Endereço
                      </Button>
                    </DialogTrigger>
                    <DialogContent className="max-w-lg">
                      <DialogHeader>
                        <DialogTitle>Selecionar Endereço de Entrega</DialogTitle>
                        <DialogDescription>
                          Escolha um endereço existente do cliente ou adicione observações sobre a entrega.
                        </DialogDescription>
                      </DialogHeader>
                      <div className="space-y-4">
                        {/* Seleção de Endereço */}
                        <div>
                          <Label htmlFor="address-select">Endereços do Cliente</Label>
                          {customerAddresses?.addresses && customerAddresses.addresses.length > 0 ? (
                            <Select
                              value={selectedAddressId}
                              onValueChange={setSelectedAddressId}
                            >
                              <SelectTrigger style={{ minHeight: '80px', textAlign: 'left' }}>
                                <SelectValue placeholder="Selecione um endereço" />
                              </SelectTrigger>
                              <SelectContent>
                                {customerAddresses.addresses.map((address) => (
                                  <SelectItem key={address.id} value={address.id}>
                                    <div className="flex flex-col">
                                      <span className="font-medium">
                                        {address.label || `${address.street}, ${address.number}`}
                                        {address.is_default && " (Padrão)"}
                                      </span>
                                      <span className="text-sm text-muted-foreground">
                                        {address.street}, {address.number} - {address.neighborhood}
                                      </span>
                                      <span className="text-sm text-muted-foreground">
                                        {address.city} - {address.state}, CEP: {address.zip_code}
                                      </span>
                                    </div>
                                  </SelectItem>
                                ))}
                              </SelectContent>
                            </Select>
                          ) : (
                            <div className="p-4 border rounded-lg text-center text-muted-foreground">
                              <MapPin className="w-8 h-8 mx-auto mb-2 opacity-50" />
                              <p>Este cliente não possui endereços cadastrados</p>
                              <p className="text-sm">Use o formulário manual abaixo</p>
                            </div>
                          )}
                        </div>

                        {/* Preview do Endereço Selecionado */}
                        {selectedAddressId && customerAddresses?.addresses && (
                          <div className="p-4 bg-muted/50 rounded-lg">
                            {(() => {
                              const selectedAddress = customerAddresses.addresses.find(addr => addr.id === selectedAddressId);
                              if (!selectedAddress) return null;
                              return (
                                <div>
                                  <h4 className="font-medium mb-2">Endereço Selecionado:</h4>
                                  <div className="text-sm space-y-1">
                                    <p><strong>Nome:</strong> {selectedAddress.name || order.customer?.name}</p>
                                    <p><strong>Endereço:</strong> {selectedAddress.street}, {selectedAddress.number}</p>
                                    {selectedAddress.complement && <p><strong>Complemento:</strong> {selectedAddress.complement}</p>}
                                    <p><strong>Bairro:</strong> {selectedAddress.neighborhood}</p>
                                    <p><strong>Cidade:</strong> {selectedAddress.city} - {selectedAddress.state}</p>
                                    <p><strong>CEP:</strong> {selectedAddress.zip_code}</p>
                                  </div>
                                </div>
                              );
                            })()}
                          </div>
                        )}

                        {/* Campo de Observações */}
                        <div>
                          <Label htmlFor="order-notes">Observações da Entrega</Label>
                          <Textarea
                            id="order-notes"
                            value={orderNotes}
                            onChange={(e) => setOrderNotes(e.target.value)}
                            placeholder="Adicione instruções especiais para entrega (ex: portão azul, interfone 101, deixar com porteiro...)"
                            rows={3}
                          />
                        </div>

                        {/* Formulário Manual (Fallback) */}
                        {(!customerAddresses?.addresses || customerAddresses.addresses.length === 0 || !selectedAddressId) && (
                          <div className="space-y-4 pt-4 border-t">
                            <h4 className="font-medium">Ou preencha manualmente:</h4>
                            <div>
                              <Label htmlFor="shipping_name">Nome do Destinatário</Label>
                              <Input
                                id="shipping_name"
                                value={shippingAddress.shipping_name}
                                onChange={(e) => setShippingAddress(prev => ({ ...prev, shipping_name: e.target.value }))}
                              />
                            </div>
                            <div className="grid grid-cols-2 gap-2">
                              <div>
                                <Label htmlFor="shipping_street">Rua</Label>
                                <Input
                                  id="shipping_street"
                                  value={shippingAddress.shipping_street}
                                  onChange={(e) => setShippingAddress(prev => ({ ...prev, shipping_street: e.target.value }))}
                                />
                              </div>
                              <div>
                                <Label htmlFor="shipping_number">Número</Label>
                                <Input
                                  id="shipping_number"
                                  value={shippingAddress.shipping_number}
                                  onChange={(e) => setShippingAddress(prev => ({ ...prev, shipping_number: e.target.value }))}
                                />
                              </div>
                            </div>
                            <div className="grid grid-cols-2 gap-2">
                              <div>
                                <Label htmlFor="shipping_complement">Complemento</Label>
                                <Input
                                  id="shipping_complement"
                                  value={shippingAddress.shipping_complement}
                                  onChange={(e) => setShippingAddress(prev => ({ ...prev, shipping_complement: e.target.value }))}
                                />
                              </div>
                              <div>
                                <Label htmlFor="shipping_neighborhood">Bairro</Label>
                                <Input
                                  id="shipping_neighborhood"
                                  value={shippingAddress.shipping_neighborhood}
                                  onChange={(e) => setShippingAddress(prev => ({ ...prev, shipping_neighborhood: e.target.value }))}
                                />
                              </div>
                            </div>
                            <div className="grid grid-cols-3 gap-2">
                              <div>
                                <Label htmlFor="shipping_city">Cidade</Label>
                                <Input
                                  id="shipping_city"
                                  value={shippingAddress.shipping_city}
                                  onChange={(e) => setShippingAddress(prev => ({ ...prev, shipping_city: e.target.value }))}
                                />
                              </div>
                              <div>
                                <Label htmlFor="shipping_state">Estado</Label>
                                <Input
                                  id="shipping_state"
                                  value={shippingAddress.shipping_state}
                                  onChange={(e) => setShippingAddress(prev => ({ ...prev, shipping_state: e.target.value }))}
                                />
                              </div>
                              <div>
                                <Label htmlFor="shipping_zipcode">CEP</Label>
                                <Input
                                  id="shipping_zipcode"
                                  value={shippingAddress.shipping_zipcode}
                                  onChange={(e) => setShippingAddress(prev => ({ ...prev, shipping_zipcode: e.target.value }))}
                                />
                              </div>
                            </div>
                          </div>
                        )}

                        <div className="flex gap-2">
                          <Button onClick={handleUpdateAddress} className="flex-1">
                            Salvar Endereço e Observações
                          </Button>
                          <Button variant="outline" onClick={() => setAddressModalOpen(false)}>
                            Cancelar
                          </Button>
                        </div>
                      </div>
                    </DialogContent>
                  </Dialog>
                )}
              </CardTitle>
            </CardHeader>
            <CardContent className="print-card-content">
              {order.shipping_street ? (
                <div className="space-y-2 print-address">
                  <p className="font-medium">{order.shipping_name || 'Nome não informado'}</p>
                  <p className="text-sm text-muted-foreground">
                    {order.shipping_street}, {order.shipping_number}
                    {order.shipping_complement && ` - ${order.shipping_complement}`}
                  </p>
                  <p className="text-sm text-muted-foreground">
                    {order.shipping_neighborhood}, {order.shipping_city} - {order.shipping_state}
                  </p>
                  <p className="text-sm text-muted-foreground">
                    CEP: {order.shipping_zipcode}
                  </p>
                </div>
              ) : (
                <div className="text-center py-4 text-muted-foreground">
                  <MapPin className="w-8 h-8 mx-auto mb-2 opacity-50" />
                  <p>Endereço de entrega não cadastrado</p>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Financial Breakdown */}
          <Card className="border-0 shadow-custom-md print-card">
            <CardHeader className="print-card-header">
              <CardTitle className="flex items-center gap-2">
                <CreditCard className="w-5 h-5 text-primary" />
                Detalhes Financeiros
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 print-card-content">
              <div className="flex justify-between print-summary">
                <span className="text-muted-foreground">Subtotal:</span>
                <span>R$ {parseFloat(order.subtotal || '0').toFixed(2).replace('.', ',')}</span>
              </div>
              {order.tax_amount && parseFloat(order.tax_amount) > 0 && (
                <div className="flex justify-between print-summary">
                  <span className="text-muted-foreground">Impostos:</span>
                  <span>R$ {parseFloat(order.tax_amount).toFixed(2).replace('.', ',')}</span>
                </div>
              )}
              {order.shipping_amount && parseFloat(order.shipping_amount) > 0 && (
                <div className="flex justify-between print-summary">
                  <span className="text-muted-foreground">Frete:</span>
                  <span>R$ {parseFloat(order.shipping_amount).toFixed(2).replace('.', ',')}</span>
                </div>
              )}
              {order.discount_amount && parseFloat(order.discount_amount) > 0 && (
                <div className="flex justify-between text-success print-summary">
                  <span>Desconto:</span>
                  <span>-R$ {parseFloat(order.discount_amount).toFixed(2).replace('.', ',')}</span>
                </div>
              )}
              <Separator />
              <div className="flex justify-between font-medium text-lg print-summary total">
                <span>Total:</span>
                <span className="text-success">
                  R$ {parseFloat(order.total_amount || '0').toFixed(2).replace('.', ',')}
                </span>
              </div>
              
              {/* Observações do Pedido */}
              {(order.observations || order.change_for) && (
                <>
                  <Separator className="mt-4" />
                  <div className="mt-4 space-y-2">
                    <h4 className="font-medium text-sm text-muted-foreground">Observações</h4>
                    {order.observations && (
                      <div className="text-sm">
                        <span className="font-medium">Obs:</span> {order.observations}
                      </div>
                    )}
                    {order.change_for && (
                      <div className="text-sm">
                        <span className="font-medium">Troco para:</span> R$ {order.change_for}
                      </div>
                    )}
                  </div>
                </>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Customer Info */}
          <Card className="border-0 shadow-custom-md print-card">
            <CardHeader className="print-card-header">
              <CardTitle className="flex items-center gap-2">
                <User className="w-5 h-5 text-primary" />
                Informações do Cliente
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4 print-card-content print-customer-info">
              <div>
                <p className="font-medium text-foreground">
                  {order.customer?.name || 'Cliente não informado'}
                </p>
                <p className="text-sm text-muted-foreground">
                  ID: {order.customer_id?.substring(0, 8) || 'N/A'}
                </p>
              </div>
              
              {order.customer?.email && (
                <div className="flex items-center gap-2">
                  <Mail className="w-4 h-4 text-muted-foreground" />
                  <span className="text-sm">{order.customer.email}</span>
                </div>
              )}

              {order.customer?.phone && (
                <div className="flex items-center gap-2">
                  <Phone className="w-4 h-4 text-muted-foreground" />
                  <span className="text-sm">{order.customer.phone}</span>
                </div>
              )}

              {/* Customer Profile Modal */}
              <Dialog open={customerModalOpen} onOpenChange={setCustomerModalOpen}>
                <DialogTrigger asChild>
                  <Button variant="outline" className="w-full no-print" size="sm">
                    <User className="w-4 h-4 mr-2" />
                    Ver Perfil do Cliente
                  </Button>
                </DialogTrigger>
                <DialogContent className="max-w-lg">
                  <DialogHeader>
                    <DialogTitle>Perfil do Cliente</DialogTitle>
                  </DialogHeader>
                  <div className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <Label>Nome</Label>
                        <p className="text-sm font-medium">{order.customer?.name || 'Nome não informado'}</p>
                      </div>
                      <div>
                        <Label>Telefone</Label>
                        <p className="text-sm">{order.customer?.phone || 'Não informado'}</p>
                      </div>
                    </div>
                    <div>
                      <Label>Email</Label>
                      <p className="text-sm">{order.customer?.email || 'Não informado'}</p>
                    </div>
                    <div>
                      <Label>ID do Cliente</Label>
                      <p className="text-sm">{order.customer_id}</p>
                    </div>
                    <div>
                      <Label>Data de Criação</Label>
                      <p className="text-sm">
                        {order.created_at ? new Date(order.created_at).toLocaleString('pt-BR') : 'N/A'}
                      </p>
                    </div>
                    <div className="flex justify-end">
                      <Button variant="outline" onClick={() => setCustomerModalOpen(false)}>
                        Fechar
                      </Button>
                    </div>
                  </div>
                </DialogContent>
              </Dialog>
            </CardContent>
          </Card>

          {/* Actions */}
          <Card className="border-0 shadow-custom-md print-card">
            <CardHeader className="print-card-header">
              <CardTitle>Ações</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 print-card-content">
              {/* Status Edit Modal */}
              <Dialog open={statusModalOpen} onOpenChange={setStatusModalOpen}>
                <DialogTrigger asChild>
                  <Button variant="outline" className="w-full no-print" size="sm">
                    <Edit className="w-4 h-4 mr-2" />
                    Editar Status
                  </Button>
                </DialogTrigger>
                <DialogContent className="max-w-md">
                  <DialogHeader>
                    <DialogTitle>Editar Status do Pedido</DialogTitle>
                  </DialogHeader>
                  <div className="space-y-4">
                    <div>
                      <Label htmlFor="status">Status do Pedido</Label>
                      <Select
                        value={newStatus}
                        onValueChange={setNewStatus}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Selecione o status" />
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

                    <div>
                      <Label htmlFor="payment-status">Status do Pagamento</Label>
                      <Select
                        value={newPaymentStatus}
                        onValueChange={setNewPaymentStatus}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Selecione o status do pagamento" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="pending">Pendente</SelectItem>
                          <SelectItem value="paid">Pago</SelectItem>
                          <SelectItem value="failed">Falhou</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>

                    <div>
                      <Label htmlFor="payment-method">Forma de Pagamento</Label>
                      <Select
                        value={newPaymentMethodId}
                        onValueChange={setNewPaymentMethodId}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Selecione a forma de pagamento" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="none">Nenhuma</SelectItem>
                          {Array.isArray(paymentMethods) && paymentMethods?.filter(method => method.id && method.id.trim() !== '').map((method: PaymentMethod) => (
                            <SelectItem key={method.id} value={method.id}>
                              {method.name}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>

                    <div>
                      <Label htmlFor="fulfillment-status">Status de Entrega</Label>
                      <Select
                        value={newFulfillmentStatus}
                        onValueChange={setNewFulfillmentStatus}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Selecione o status de entrega" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="pending">Pendente</SelectItem>
                          <SelectItem value="shipped">Enviado</SelectItem>
                          <SelectItem value="delivered">Entregue</SelectItem>
                          <SelectItem value="cancelled">Cancelado</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>

                    <div>
                      <Label htmlFor="discount">Desconto (R$)</Label>
                      <Input
                        id="discount"
                        type="number"
                        step="0.01"
                        min="0"
                        max={parseFloat(order.total_amount || '0')}
                        value={discountAmount}
                        onChange={(e) => setDiscountAmount(e.target.value)}
                        placeholder="0.00"
                      />
                    </div>
                    <div>
                      <Label>Valor Final</Label>
                      <p className="text-lg font-semibold">
                        R$ {(parseFloat(order.total_amount || '0') - parseFloat(discountAmount || '0')).toFixed(2)}
                      </p>
                    </div>
                    <div className="flex gap-2">
                      <Button onClick={handleUpdateStatus} className="flex-1">
                        Salvar Alterações
                      </Button>
                      <Button variant="outline" onClick={() => setStatusModalOpen(false)}>
                        Cancelar
                      </Button>
                    </div>
                  </div>
                </DialogContent>
              </Dialog>

              {/* Generate PDF Button */}
              <Button 
                variant="outline" 
                className="w-full no-print" 
                size="sm"
                onClick={handleGeneratePDF}
              >
                <FileText className="w-4 h-4 mr-2" />
                Gerar PDF
              </Button>

              {/* Email Modal */}
              <Dialog open={emailModalOpen} onOpenChange={setEmailModalOpen}>
                <DialogTrigger asChild>
                  <Button variant="outline" className="w-full no-print" size="sm">
                    <Send className="w-4 h-4 mr-2" />
                    Enviar por Email
                  </Button>
                </DialogTrigger>
                <DialogContent className="max-w-lg">
                  <DialogHeader>
                    <DialogTitle>Enviar Pedido por Email</DialogTitle>
                  </DialogHeader>
                  <div className="space-y-4">
                    <div>
                      <Label htmlFor="email-recipient">Email do Destinatário</Label>
                      <Input
                        id="email-recipient"
                        type="email"
                        value={emailRecipient}
                        onChange={(e) => setEmailRecipient(e.target.value)}
                        placeholder="email@exemplo.com"
                      />
                    </div>
                    <div>
                      <Label htmlFor="email-subject">Assunto</Label>
                      <Input
                        id="email-subject"
                        value={emailSubject}
                        onChange={(e) => setEmailSubject(e.target.value)}
                        placeholder="Detalhes do seu pedido"
                      />
                    </div>
                    <div>
                      <Label htmlFor="email-message">Mensagem</Label>
                      <textarea
                        id="email-message"
                        className="w-full min-h-[100px] p-3 border border-input bg-background text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 rounded-md"
                        value={emailMessage}
                        onChange={(e) => setEmailMessage(e.target.value)}
                        placeholder="Adicione uma mensagem personalizada..."
                      />
                    </div>
                    <div className="flex gap-2">
                      <Button 
                        onClick={handleSendEmail} 
                        className="flex-1"
                        disabled={emailSending || !emailRecipient.trim()}
                      >
                        {emailSending ? 'Enviando...' : 'Enviar Email'}
                      </Button>
                      <Button variant="outline" onClick={() => setEmailModalOpen(false)}>
                        Cancelar
                      </Button>
                    </div>
                  </div>
                </DialogContent>
              </Dialog>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
