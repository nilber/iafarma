import { parsePhoneNumber } from 'libphonenumber-js';

// Função para formatar telefone para exibição na lista
export function formatPhoneForDisplay(phone: string): string {
  if (!phone) return '';
  
  try {
    // Remove @c.us se existir (formato WhatsApp)
    const cleanPhone = phone.replace('@c.us', '');
    
    // Se já tem + no início, tenta fazer parse direto
    if (cleanPhone.startsWith('+')) {
      const phoneNumber = parsePhoneNumber(cleanPhone);
      return phoneNumber ? phoneNumber.formatInternational() : cleanPhone;
    }
    
    // Se começa com 55 (Brasil), adiciona +
    if (cleanPhone.startsWith('55') && cleanPhone.length >= 12) {
      const phoneNumber = parsePhoneNumber('+' + cleanPhone);
      return phoneNumber ? phoneNumber.formatInternational() : cleanPhone;
    }
    
    // Para números brasileiros sem código de país
    const cleaned = cleanPhone.replace(/\D/g, '');
    if (cleaned.length === 11) {
      // Celular brasileiro
      return `(${cleaned.substring(0, 2)}) ${cleaned.substring(2, 7)}-${cleaned.substring(7)}`;
    } else if (cleaned.length === 10) {
      // Telefone fixo brasileiro
      return `(${cleaned.substring(0, 2)}) ${cleaned.substring(2, 6)}-${cleaned.substring(6)}`;
    }
    
    return phone;
  } catch (error) {
    // Se não conseguir fazer parse, retorna o telefone original
    return phone;
  }
}

// Função para limpar telefone para salvar no banco (apenas números)
export function cleanPhoneForStorage(phone: string): string {
  if (!phone) return '';
  return phone.replace(/\D/g, '');
}

// Função para converter telefone para formato WhatsApp
export function formatPhoneForWhatsApp(phone: string): string {
  if (!phone) return '';
  
  // Remove all non-digit characters
  const digitsOnly = phone.replace(/\D/g, '');
  
  // If it already starts with 55 (Brazil), use as is
  // If it doesn't have country code, add 55 (Brazil)
  let formattedNumber = digitsOnly;
  
  if (formattedNumber.startsWith('0')) {
    formattedNumber = '55' + formattedNumber.slice(1);
  } else if (!formattedNumber.startsWith('55') && formattedNumber.length >= 10) {
    formattedNumber = '55' + formattedNumber;
  }
  
  return formattedNumber + '@c.us';
}

// Função para converter de formato WhatsApp para formato de exibição no input
export function formatPhoneFromWhatsApp(phone: string): string {
  if (!phone) return '';
  
  // Remove @c.us
  const cleanPhone = phone.replace('@c.us', '');
  
  // Se começa com 55, adiciona +
  if (cleanPhone.startsWith('55')) {
    return '+' + cleanPhone;
  }
  
  return cleanPhone;
}
