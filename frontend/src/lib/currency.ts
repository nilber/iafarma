/**
 * Formats a number as currency (Brazilian Real)
 */
export function formatCurrency(value: number): string {
  return new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency: 'BRL'
  }).format(value);
}

/**
 * Parses a currency string to number
 */
export function parseCurrency(value: string): number {
  // Remove currency symbols and convert to number
  const numericValue = value
    .replace(/[^\d,.-]/g, '') // Remove non-numeric characters except comma, dot, dash
    .replace(/\./g, '') // Remove thousands separators
    .replace(',', '.'); // Convert decimal comma to dot
  
  return parseFloat(numericValue) || 0;
}

/**
 * Formats a number input value for currency display
 */
export function formatCurrencyInput(value: string): string {
  // Remove all non-numeric characters except comma and dot
  const numericValue = value.replace(/[^\d]/g, '');
  
  if (!numericValue) return '';
  
  // Convert to number and format
  const number = parseFloat(numericValue) / 100;
  
  return new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency: 'BRL'
  }).format(number);
}

/**
 * Formats a value for currency input (removes currency symbols for API)
 */
export function formatForAPI(value: string): string {
  if (!value) return '';
  
  // Extract numeric value and convert to decimal string
  const numericValue = value.replace(/[^\d]/g, '');
  if (!numericValue) return '';
  
  const number = parseFloat(numericValue) / 100;
  return number.toFixed(2);
}

/**
 * Masks input for currency with real-time formatting
 */
export function applyCurrencyMask(value: string): string {
  // Remove all non-numeric characters
  const numbers = value.replace(/\D/g, '');
  
  if (!numbers) return '';
  
  // Convert to cents and format
  const amount = parseInt(numbers, 10) / 100;
  
  return new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency: 'BRL',
    minimumFractionDigits: 2
  }).format(amount);
}
