import { Check, CheckCheck, Clock, AlertCircle } from "lucide-react";
import { cn } from "@/lib/utils";

interface MessageStatusIconProps {
  status?: string;
  className?: string;
}

export function MessageStatusIcon({ status, className }: MessageStatusIconProps) {
  switch (status) {
    case 'sent':
      // 1 check cinza - mensagem enviada
      return (
        <Check 
          className={cn("w-3.5 h-3.5 text-gray-400", className)} 
          strokeWidth={2}
        />
      );
    case 'delivered':
      // 2 checks cinza - mensagem entregue
      return (
        <CheckCheck 
          className={cn("w-3.5 h-3.5 text-gray-400", className)} 
          strokeWidth={2}
        />
      );
    case 'read':
      // 2 checks azul - mensagem lida
      return (
        <CheckCheck 
          className={cn("w-3.5 h-3.5 text-blue-500", className)} 
          strokeWidth={2}
        />
      );
    case 'sending':
      // Relógio - enviando
      return (
        <Clock 
          className={cn("w-3.5 h-3.5 text-gray-400", className)} 
          strokeWidth={2}
        />
      );
    case 'failed':
      // Círculo vermelho - falha
      return (
        <AlertCircle 
          className={cn("w-3.5 h-3.5 text-red-500", className)} 
          strokeWidth={2}
        />
      );
    default:
      return null;
  }
}
