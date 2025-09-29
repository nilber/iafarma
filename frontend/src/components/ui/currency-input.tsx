import React, { useState, useEffect } from 'react';
import { Input } from '@/components/ui/input';

interface CurrencyInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
  disabled?: boolean;
}

export function CurrencyInput({ 
  value, 
  onChange, 
  placeholder = "R$ 0,00", 
  className,
  disabled 
}: CurrencyInputProps) {
  const [displayValue, setDisplayValue] = useState('');

  // Função para aplicar máscara de moeda
  const applyCurrencyMask = (inputValue: string): string => {
    // Remove tudo que não é dígito
    const digits = inputValue.replace(/\D/g, '');
    
    if (!digits) return '';
    
    // Converte para número (centavos)
    const number = parseInt(digits);
    
    // Converte centavos para reais
    const reais = number / 100;
    
    // Formata como moeda brasileira
    return reais.toLocaleString('pt-BR', {
      style: 'currency',
      currency: 'BRL',
      minimumFractionDigits: 2,
      maximumFractionDigits: 2
    });
  };

  // Função para converter display value para API format
  const formatForAPI = (displayValue: string): string => {
    // Remove tudo que não é dígito ou vírgula
    const cleaned = displayValue.replace(/[^\d,]/g, '');
    
    if (!cleaned) return '';
    
    // Substitui vírgula por ponto para formato API
    return cleaned.replace(',', '.');
  };

  // Initialize display value when value prop changes
  useEffect(() => {
    if (value) {
      // Se o valor já está formatado, usa ele; senão formata
      if (value.includes('R$')) {
        setDisplayValue(value);
      } else {
        // Converte valor da API (e.g., "10.50") para formato display (e.g., "R$ 10,50")
        const numericValue = parseFloat(value) || 0;
        const formatted = numericValue.toLocaleString('pt-BR', {
          style: 'currency',
          currency: 'BRL',
          minimumFractionDigits: 2,
          maximumFractionDigits: 2
        });
        setDisplayValue(formatted);
      }
    } else {
      setDisplayValue('');
    }
  }, [value]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const inputValue = e.target.value;
    const masked = applyCurrencyMask(inputValue);
    setDisplayValue(masked);
    
    // Converte de volta para formato API
    const apiValue = formatForAPI(masked);
    onChange(apiValue);
  };

  return (
    <Input
      type="text"
      value={displayValue}
      onChange={handleChange}
      placeholder={placeholder}
      className={className}
      disabled={disabled}
    />
  );
}
