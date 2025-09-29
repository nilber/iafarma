import { useState } from "react";
import { Upload, Download, FileText, AlertCircle, CheckCircle, XCircle } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Progress } from "@/components/ui/progress";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useImportProducts, useDownloadImportTemplate, queryKeys } from "@/lib/api/hooks";
import { ProductImportItem, ProductImportResult } from "@/lib/api/types";
import { toast } from "sonner";
import { useQueryClient } from "@tanstack/react-query";

interface ProductImportDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ProductImportDialog({ open, onOpenChange }: ProductImportDialogProps) {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [importResult, setImportResult] = useState<ProductImportResult | null>(null);
  const [isProcessing, setIsProcessing] = useState(false);

  const queryClient = useQueryClient();
  const importProducts = useImportProducts();
  const downloadTemplate = useDownloadImportTemplate();

  const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file && (file.type === 'text/csv' || file.name.endsWith('.csv'))) {
      setSelectedFile(file);
      setImportResult(null);
    } else {
      toast.error('Por favor, selecione um arquivo CSV válido');
    }
  };

  const handleDownloadTemplate = async () => {
    try {
      const csvContent = await downloadTemplate.mutateAsync();
      const blob = new Blob([csvContent], { type: 'text/csv' });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'template_importacao_produtos.csv';
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      window.URL.revokeObjectURL(url);
      toast.success('Template baixado com sucesso');
    } catch (error) {
      toast.error('Erro ao baixar o template');
    }
  };

  const parseCSV = (csvContent: string): ProductImportItem[] => {
    const lines = csvContent.split('\n').filter(line => line.trim());
    if (lines.length < 2) return [];

    // Detect separator (comma or semicolon)
    const firstLine = lines[0];
    const commaCount = (firstLine.match(/,/g) || []).length;
    const semicolonCount = (firstLine.match(/;/g) || []).length;
    const separator = semicolonCount > commaCount ? ';' : ',';

    // Parse CSV properly handling quotes and separators
    const parseCSVLine = (line: string): string[] => {
      const values: string[] = [];
      let current = '';
      let inQuotes = false;
      let i = 0;

      while (i < line.length) {
        const char = line[i];
        const nextChar = line[i + 1];

        if (char === '"') {
          if (inQuotes && nextChar === '"') {
            // Escaped quote inside quoted field
            current += '"';
            i += 2;
          } else {
            // Start or end of quoted field
            inQuotes = !inQuotes;
            i++;
          }
        } else if (char === separator && !inQuotes) {
          // Field separator
          values.push(current.trim());
          current = '';
          i++;
        } else {
          current += char;
          i++;
        }
      }

      // Add the last field
      values.push(current.trim());
      return values;
    };

    const headers = parseCSVLine(lines[0]).map(h => h.toLowerCase().replace(/"/g, '').trim());
    const products: ProductImportItem[] = [];

    for (let i = 1; i < lines.length; i++) {
      const values = parseCSVLine(lines[i]).map(v => v.replace(/^"|"$/g, '').trim());
      if (values.length < headers.length) continue;

      const getValueByHeader = (headerName: string): string => {
        const index = headers.indexOf(headerName);
        return index >= 0 ? values[index] || '' : '';
      };

      // Normalize price values (replace comma with dot for decimal separator)
      const normalizePrice = (priceStr: string): string => {
        if (!priceStr) return '';
        return priceStr.replace(',', '.');
      };

      const product: ProductImportItem = {
        name: getValueByHeader('name'),
        description: getValueByHeader('description'),
        price: normalizePrice(getValueByHeader('price')),
        sale_price: normalizePrice(getValueByHeader('sale_price')),
        sku: getValueByHeader('sku'),
        barcode: getValueByHeader('barcode'),
        weight: getValueByHeader('weight'),
        dimensions: getValueByHeader('dimensions'),
        brand: getValueByHeader('brand'),
        tags: getValueByHeader('tags'),
        stock_quantity: parseInt(getValueByHeader('stock_quantity') || '0'),
        low_stock_threshold: parseInt(getValueByHeader('low_stock_threshold') || '5'),
        category_name: getValueByHeader('category_name') || getValueByHeader('category'),
      };

      if (product.name && product.price) {
        products.push(product);
      }
    }

    return products;
  };

  const handleImport = async () => {
    if (!selectedFile) return;

    setIsProcessing(true);
    try {
      const csvContent = await selectedFile.text();
      const products = parseCSV(csvContent);

      if (products.length === 0) {
        toast.error('Nenhum produto válido encontrado no arquivo');
        return;
      }

      const result = await importProducts.mutateAsync({ products });
      setImportResult(result);

      const { created, updated, errors } = result;
      if (errors === 0) {
        toast.success(`Importação concluída! ${created} criados, ${updated} atualizados`);
      } else {
        toast.warning(`Importação concluída com ${errors} erro(s). ${created} criados, ${updated} atualizados`);
      }
    } catch (error) {
      // Extract error message from API response
      let errorMessage = 'Erro ao processar o arquivo';
      
      if (error instanceof Error && error.message) {
        errorMessage = error.message;
      }
      
      toast.error(errorMessage);
    } finally {
      setIsProcessing(false);
    }
  };

  const resetDialog = () => {
    setSelectedFile(null);
    setImportResult(null);
    setIsProcessing(false);
  };

  const handleClose = () => {
    // If there was a successful import, force refresh the products list
    if (importResult && (importResult.created > 0 || importResult.updated > 0)) {
      queryClient.invalidateQueries({ queryKey: queryKeys.products });
    }
    
    resetDialog();
    onOpenChange(false);
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'created':
      case 'updated':
        return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'error':
        return <XCircle className="w-4 h-4 text-red-500" />;
      default:
        return <AlertCircle className="w-4 h-4 text-yellow-500" />;
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'created':
        return <Badge variant="default" className="bg-green-100 text-green-800">Criado</Badge>;
      case 'updated':
        return <Badge variant="default" className="bg-blue-100 text-blue-800">Atualizado</Badge>;
      case 'error':
        return <Badge variant="destructive">Erro</Badge>;
      default:
        return <Badge variant="secondary">{status}</Badge>;
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Importar Produtos</DialogTitle>
          <DialogDescription>
            Importe produtos em massa através de um arquivo CSV
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {/* Template Download */}
          <div className="border rounded-lg p-4 bg-blue-50">
            <div className="flex items-start gap-3">
              <FileText className="w-5 h-5 text-blue-600 mt-0.5" />
              <div className="flex-1">
                <h4 className="font-medium text-blue-900">Template de Importação</h4>
                <p className="text-sm text-blue-700 mt-1">
                  Baixe o template CSV com exemplos para preencher com seus produtos
                </p>
              </div>
              <Button 
                variant="outline" 
                size="sm"
                onClick={handleDownloadTemplate}
                disabled={downloadTemplate.isPending}
              >
                <Download className="w-4 h-4 mr-2" />
                Baixar Template
              </Button>
            </div>
          </div>

          {/* File Upload */}
          <div className="space-y-2">
            <Label htmlFor="csv-file">Arquivo CSV</Label>
            <Input
              id="csv-file"
              type="file"
              accept=".csv"
              onChange={handleFileSelect}
              disabled={isProcessing}
            />
            {selectedFile && (
              <p className="text-sm text-muted-foreground">
                Arquivo selecionado: {selectedFile.name}
              </p>
            )}
          </div>

          {/* Import Button */}
          <div className="flex gap-2">
            <Button
              onClick={handleImport}
              disabled={!selectedFile || isProcessing}
              className="flex-1"
            >
              {isProcessing ? (
                <>
                  <Progress className="w-4 h-4 mr-2" />
                  Processando...
                </>
              ) : (
                <>
                  <Upload className="w-4 h-4 mr-2" />
                  Importar Produtos
                </>
              )}
            </Button>
            <Button variant="outline" onClick={handleClose}>
              {importResult ? 'Fechar' : 'Cancelar'}
            </Button>
          </div>

          {/* Import Results */}
          {importResult && (
            <div className="space-y-4">
              <div className="border rounded-lg p-4">
                <h4 className="font-medium mb-3">Resultado da Importação</h4>
                <div className="grid grid-cols-4 gap-4 text-sm">
                  <div className="text-center">
                    <div className="text-2xl font-bold text-blue-600">
                      {importResult.total_processed}
                    </div>
                    <div className="text-muted-foreground">Total</div>
                  </div>
                  <div className="text-center">
                    <div className="text-2xl font-bold text-green-600">
                      {importResult.created}
                    </div>
                    <div className="text-muted-foreground">Criados</div>
                  </div>
                  <div className="text-center">
                    <div className="text-2xl font-bold text-blue-600">
                      {importResult.updated}
                    </div>
                    <div className="text-muted-foreground">Atualizados</div>
                  </div>
                  <div className="text-center">
                    <div className="text-2xl font-bold text-red-600">
                      {importResult.errors}
                    </div>
                    <div className="text-muted-foreground">Erros</div>
                  </div>
                </div>
              </div>

              {/* Detailed Results */}
              <div className="border rounded-lg">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-16">Linha</TableHead>
                      <TableHead>Nome</TableHead>
                      <TableHead>SKU</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Mensagem</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {importResult.results.map((result) => (
                      <TableRow key={result.row_number}>
                        <TableCell>{result.row_number}</TableCell>
                        <TableCell className="font-medium">{result.name}</TableCell>
                        <TableCell>{result.sku || '-'}</TableCell>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            {getStatusIcon(result.status)}
                            {getStatusBadge(result.status)}
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="text-sm">
                            <div>{result.message}</div>
                            {result.error && (
                              <div className="text-red-600 mt-1">{result.error}</div>
                            )}
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
