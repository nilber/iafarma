import { Badge } from "@/components/ui/badge";
import { AlertTriangle, Package, XCircle } from "lucide-react";
import { cn } from "@/lib/utils";

interface StockStatusBadgeProps {
  stockQuantity: number | undefined;
  lowStockThreshold: number | undefined;
}

export function StockStatusBadge({ stockQuantity = 0, lowStockThreshold = 5 }: StockStatusBadgeProps) {
  const stock = stockQuantity || 0;
  const threshold = lowStockThreshold || 5;

  if (stock === 0) {
    return (
      <div className="flex items-center gap-2">
        <Badge 
          variant="destructive" 
          className="flex items-center gap-1 bg-red-100 text-red-800 border-red-200 hover:bg-red-200"
        >
          <XCircle className="w-3 h-3" />
          Sem Estoque
        </Badge>
        <span className="text-sm text-red-600 font-medium">0 unidades</span>
      </div>
    );
  }

  if (stock <= threshold) {
    return (
      <div className="flex items-center gap-2">
        <Badge 
          variant="secondary" 
          className="flex items-center gap-1 bg-amber-100 text-amber-800 border-amber-200 hover:bg-amber-200"
        >
          <AlertTriangle className="w-3 h-3" />
          Baixo
        </Badge>
        <span className="text-sm text-amber-700 font-medium">{stock} unidades</span>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2">
      <Badge 
        variant="secondary" 
        className="flex items-center gap-1 bg-emerald-100 text-emerald-800 border-emerald-200 hover:bg-emerald-200"
      >
        <Package className="w-3 h-3" />
        OK
      </Badge>
      <span className="text-sm text-emerald-700 font-medium">{stock} unidades</span>
    </div>
  );
}