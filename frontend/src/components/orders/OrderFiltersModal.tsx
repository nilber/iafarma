import { useState, useEffect } from 'react';
import { Filter, X, CalendarIcon } from "lucide-react";
import { format } from "date-fns";
import { ptBR } from "date-fns/locale";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Calendar } from "@/components/ui/calendar";
import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api/client";

export interface OrderFilters {
  search?: string;
  status?: string;
  payment_status?: string;
  fulfillment_status?: string;
  payment_method_id?: string;
  customer_id?: string;
  date_from?: string;
  date_to?: string;
}

interface OrderFiltersModalProps {
  filters: OrderFilters;
  onFiltersChange: (filters: OrderFilters) => void;
}

export function OrderFiltersModal({ filters, onFiltersChange }: OrderFiltersModalProps) {
  const [open, setOpen] = useState(false);
  const [localFilters, setLocalFilters] = useState<OrderFilters>(filters);
  const [dateFrom, setDateFrom] = useState<Date | undefined>(
    filters.date_from ? new Date(filters.date_from) : undefined
  );
  const [dateTo, setDateTo] = useState<Date | undefined>(
    filters.date_to ? new Date(filters.date_to) : undefined
  );

  // Fetch payment methods for filter
  const { data: paymentMethods } = useQuery({
    queryKey: ['payment-methods-active'],
    queryFn: () => apiClient.getActivePaymentMethods(),
  });

  // Sync local filters when external filters change
  useEffect(() => {
    setLocalFilters(filters);
    setDateFrom(filters.date_from ? new Date(filters.date_from) : undefined);
    setDateTo(filters.date_to ? new Date(filters.date_to) : undefined);
  }, [filters]);

  const handleApplyFilters = () => {
    const updatedFilters = {
      ...localFilters,
      date_from: dateFrom ? format(dateFrom, 'yyyy-MM-dd') : undefined,
      date_to: dateTo ? format(dateTo, 'yyyy-MM-dd') : undefined,
    };
    onFiltersChange(updatedFilters);
    setOpen(false);
  };

  const handleClearFilters = () => {
    const clearedFilters: OrderFilters = {};
    setLocalFilters(clearedFilters);
    setDateFrom(undefined);
    setDateTo(undefined);
    onFiltersChange(clearedFilters);
    setOpen(false);
  };

  const hasActiveFilters = Object.values(filters).some(value => 
    value !== undefined && value !== ""
  );

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm" className={cn(hasActiveFilters && "border-primary")}>
          <Filter className="w-4 h-4 mr-2" />
          Filtros
          {hasActiveFilters && (
            <span className="ml-1 px-2 py-0.5 text-xs bg-primary text-primary-foreground rounded-full">
              {Object.values(filters).filter(v => v !== undefined && v !== "").length}
            </span>
          )}
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Filtros Avançados</DialogTitle>
          <DialogDescription>
            Configure filtros específicos para encontrar os pedidos desejados
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-6 py-4">
          {/* Busca por texto */}
          <div className="grid gap-2">
            <Label htmlFor="search">Buscar por pedido, cliente, telefone ou email</Label>
            <Input
              id="search"
              placeholder="Digite o termo de busca..."
              value={localFilters.search || ""}
              onChange={(e) => setLocalFilters({...localFilters, search: e.target.value})}
            />
          </div>

          {/* Status do pedido */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="grid gap-2">
              <Label>Status do Pedido</Label>
              <Select
                value={localFilters.status || "all"}
                onValueChange={(value) => setLocalFilters({...localFilters, status: value === "all" ? undefined : value})}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Todos os status" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Todos os status</SelectItem>
                  <SelectItem value="pending">Pendente</SelectItem>
                  <SelectItem value="confirmed">Confirmado</SelectItem>
                  <SelectItem value="processing">Processando</SelectItem>
                  <SelectItem value="shipped">Enviado</SelectItem>
                  <SelectItem value="delivered">Entregue</SelectItem>
                  <SelectItem value="cancelled">Cancelado</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="grid gap-2">
              <Label>Status do Pagamento</Label>
              <Select
                value={localFilters.payment_status || "all"}
                onValueChange={(value) => setLocalFilters({...localFilters, payment_status: value === "all" ? undefined : value})}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Todos" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Todos</SelectItem>
                  <SelectItem value="pending">Pendente</SelectItem>
                  <SelectItem value="paid">Pago</SelectItem>
                  <SelectItem value="failed">Falhou</SelectItem>
                  <SelectItem value="refunded">Reembolsado</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="grid gap-2">
              <Label>Status de Entrega</Label>
              <Select
                value={localFilters.fulfillment_status || "all"}
                onValueChange={(value) => setLocalFilters({...localFilters, fulfillment_status: value === "all" ? undefined : value})}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Todos" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Todos</SelectItem>
                  <SelectItem value="pending">Pendente</SelectItem>
                  <SelectItem value="processing">Processando</SelectItem>
                  <SelectItem value="shipped">Enviado</SelectItem>
                  <SelectItem value="delivered">Entregue</SelectItem>
                  <SelectItem value="cancelled">Cancelado</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Método de pagamento */}
          <div className="grid gap-2">
            <Label>Método de Pagamento</Label>
            <Select
              value={localFilters.payment_method_id || "all"}
              onValueChange={(value) => setLocalFilters({...localFilters, payment_method_id: value === "all" ? undefined : value})}
            >
              <SelectTrigger>
                <SelectValue placeholder="Todos os métodos" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Todos os métodos</SelectItem>
                {paymentMethods?.map((method) => (
                  <SelectItem key={method.id} value={method.id}>
                    {method.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Filtros de data */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="grid gap-2">
              <Label>Data inicial</Label>
              <Popover>
                <PopoverTrigger asChild>
                  <Button
                    variant="outline"
                    className={cn(
                      "w-full justify-start text-left font-normal",
                      !dateFrom && "text-muted-foreground"
                    )}
                  >
                    <CalendarIcon className="mr-2 h-4 w-4" />
                    {dateFrom ? format(dateFrom, "PPP", { locale: ptBR }) : "Selecione..."}
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-auto p-0" align="start">
                  <Calendar
                    mode="single"
                    selected={dateFrom}
                    onSelect={setDateFrom}
                    initialFocus
                  />
                </PopoverContent>
              </Popover>
            </div>

            <div className="grid gap-2">
              <Label>Data final</Label>
              <Popover>
                <PopoverTrigger asChild>
                  <Button
                    variant="outline"
                    className={cn(
                      "w-full justify-start text-left font-normal",
                      !dateTo && "text-muted-foreground"
                    )}
                  >
                    <CalendarIcon className="mr-2 h-4 w-4" />
                    {dateTo ? format(dateTo, "PPP", { locale: ptBR }) : "Selecione..."}
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-auto p-0" align="start">
                  <Calendar
                    mode="single"
                    selected={dateTo}
                    onSelect={setDateTo}
                    initialFocus
                  />
                </PopoverContent>
              </Popover>
            </div>
          </div>
        </div>

        <DialogFooter className="flex gap-2">
          <Button variant="outline" onClick={handleClearFilters}>
            Limpar Filtros
          </Button>
          <Button onClick={handleApplyFilters}>
            Aplicar Filtros
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}