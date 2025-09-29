import React from 'react';
import PhoneInput from 'react-phone-number-input';
import { cn } from '@/lib/utils';
import 'react-phone-number-input/style.css';

interface PhoneNumberInputProps {
  value?: string;
  onChange?: (value?: string) => void;
  placeholder?: string;
  className?: string;
  disabled?: boolean;
  error?: boolean;
}

export const PhoneNumberInput = React.forwardRef<
  HTMLInputElement,
  PhoneNumberInputProps
>(({ value, onChange, placeholder = "Digite o número do telefone", className, disabled, error, ...props }, ref) => {
  const handlePhoneChange = (phoneNumber?: string) => {
    if (onChange) {
      // Apenas passa o número como está (será limpo na função cleanPhoneForStorage)
      onChange(phoneNumber);
    }
  };

  // Convert WhatsApp format back to display format for editing
  const displayValue = value 
    ? value.replace('@c.us', '').replace(/^55/, '+55')
    : '';

  return (
    <div className={cn("relative", className)}>
      <PhoneInput
        international
        defaultCountry="BR"
        value={displayValue}
        onChange={handlePhoneChange}
        placeholder={placeholder}
        disabled={disabled}
        className={cn(
          "flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
          error && "border-destructive focus-visible:ring-destructive",
        )}
        numberInputProps={{
          className: cn(
            "flex h-10 w-full rounded-md border-0 bg-transparent px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50",
          ),
          ref,
          ...props
        }}
      />
    </div>
  );
});

PhoneNumberInput.displayName = "PhoneNumberInput";
