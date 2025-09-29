import { useState, useRef, useEffect } from "react";
import { Upload, Image, FileText, AlertCircle, CheckCircle, XCircle, Loader2, CreditCard } from "lucide-react";
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
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useImportProductsFromImage } from "@/lib/api/hooks";
import { ProductImageImportResult } from "@/lib/api/types";
import { toast } from "sonner";

interface ProductImageImportDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ProductImageImportDialog({ open, onOpenChange }: ProductImageImportDialogProps) {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<string | null>(null);
  const [importResult, setImportResult] = useState<ProductImageImportResult | null>(null);
  const [isProcessing, setIsProcessing] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const importProducts = useImportProductsFromImage();

  // Reset state when modal is closed
  useEffect(() => {
    if (!open) {
      handleReset();
    }
  }, [open]);

  const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      const validTypes = ['image/jpeg', 'image/jpg', 'image/png', 'image/webp', 'application/pdf'];
      if (validTypes.includes(file.type)) {
        setSelectedFile(file);
        setImportResult(null);
        
        // Create preview for images
        if (file.type.startsWith('image/')) {
          const reader = new FileReader();
          reader.onload = (e) => {
            setPreview(e.target?.result as string);
          };
          reader.readAsDataURL(file);
        } else {
          setPreview(null);
        }
      } else {
        toast.error('Por favor, selecione uma imagem (JPG, PNG, WebP) ou PDF válido');
      }
    }
  };

  const handleImport = async () => {
    if (!selectedFile) {
      toast.error('Selecione um arquivo para importar');
      return;
    }

    setIsProcessing(true);
    try {
      const formData = new FormData();
      formData.append('file', selectedFile);

      const result = await importProducts.mutateAsync(formData);
      setImportResult(result);
      
      if (result.success && result.created_count > 0) {
        toast.success(`${result.created_count} produtos criados com sucesso!`);
        // Auto-close dialog after successful import
        setTimeout(() => {
          onOpenChange(false);
          handleReset();
        }, 2000);
      } else if (result.errors && result.errors.length > 0) {
        toast.error(`Importação concluída com ${result.errors.length} erros`);
      }
    } catch (error: any) {
      console.error('Import error:', error);
      console.log('Error data:', error?.data);
      console.log('Error status:', error?.status);
      
      // Get error data from different possible sources
      const errorData = error?.data || error?.response?.data || error;
      
      // Handle insufficient credits error specifically (status 402)
      if (error?.status === 402 || errorData?.error === "Insufficient AI credits") {
        if (errorData?.available !== undefined && errorData?.required !== undefined) {
          toast.error(
            `Créditos insuficientes! Você tem ${errorData.available} créditos disponíveis, mas precisa de ${errorData.required} créditos para esta operação.`,
            { duration: 8000 }
          );
        } else {
          toast.error('Créditos insuficientes. Você precisa de 1000 créditos para esta operação.', { duration: 6000 });
        }
      } else if (errorData?.error?.includes("failed to parse AI response as JSON") || error?.message?.includes("failed to parse AI response as JSON")) {
        toast.error('Erro ao processar resposta da IA. A imagem pode ser muito complexa ou conter muitos produtos. Tente uma imagem mais simples.', { duration: 8000 });
      } else if (errorData?.error?.includes("unexpected end of JSON input") || error?.message?.includes("unexpected end of JSON input")) {
        toast.error('Erro no processamento da resposta da IA. Tente novamente com uma imagem mais simples.', { duration: 8000 });
      } else {
        const errorMessage = errorData?.error || error?.message || 'Erro ao importar produtos da imagem';
        toast.error(errorMessage, { duration: 6000 });
      }
    } finally {
      setIsProcessing(false);
    }
  };

  const handleReset = () => {
    setSelectedFile(null);
    setPreview(null);
    setImportResult(null);
    setIsProcessing(false);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const handleClose = () => {
    // Prevent closing during processing
    if (isProcessing) {
      toast.warning('Por favor, aguarde a conclusão da importação antes de fechar.');
      return;
    }
    handleReset();
    onOpenChange(false);
  };

  // Custom handler for dialog onOpenChange
  const handleDialogOpenChange = (open: boolean) => {
    if (!open) {
      handleClose();
    } else {
      onOpenChange(open);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'success':
        return <CheckCircle className="w-4 h-4 text-green-600" />;
      case 'error':
        return <XCircle className="w-4 h-4 text-red-600" />;
      default:
        return <AlertCircle className="w-4 h-4 text-yellow-600" />;
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'success':
        return <Badge variant="default" className="bg-green-100 text-green-800">Criado</Badge>;
      case 'error':
        return <Badge variant="destructive">Erro</Badge>;
      default:
        return <Badge variant="secondary">Processando</Badge>;
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleDialogOpenChange}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Image className="w-5 h-5" />
            Importação com Imagem
          </DialogTitle>
          <DialogDescription>
            Faça upload de um cardápio em imagem ou PDF para extrair produtos automaticamente usando IA.
            <span className="block mt-1 text-yellow-600 font-medium">
              💰 Custo: 1000 créditos por importação
            </span>
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {/* Credit Warning */}
          <Card className="border-yellow-200 bg-yellow-50">
            <CardHeader className="pb-3">
              <CardTitle className="text-sm flex items-center gap-2 text-yellow-800">
                <CreditCard className="w-4 h-4" />
                Informações Importantes
              </CardTitle>
            </CardHeader>
            <CardContent className="pt-0">
              <ul className="text-sm text-yellow-700 space-y-1">
                <li>• Esta funcionalidade utiliza IA para analisar cardápios e extrair informações de produtos</li>
                <li>• Custo: 1000 créditos por importação</li>
                <li>• Formatos aceitos: JPG, PNG, WebP, PDF</li>
                <li>• Estoque será definido automaticamente como 10.000 unidades</li>
                <li>• Revise os produtos antes de finalizar a importação</li>
              </ul>
            </CardContent>
          </Card>

          {/* File Upload */}
          <div className="space-y-4">
            <Label htmlFor="image-upload">Selecionar Arquivo</Label>
            <div className="border-2 border-dashed border-gray-300 rounded-lg p-6">
              <div className="text-center">
                {preview ? (
                  <div className="space-y-4">
                    <img 
                      src={preview} 
                      alt="Preview" 
                      className="max-w-full max-h-64 mx-auto rounded-lg shadow-md"
                    />
                    <p className="text-sm text-muted-foreground">
                      {selectedFile?.name} ({((selectedFile?.size || 0) / 1024 / 1024).toFixed(2)} MB)
                    </p>
                  </div>
                ) : selectedFile?.type === 'application/pdf' ? (
                  <div className="space-y-4">
                    <FileText className="w-16 h-16 text-gray-400 mx-auto" />
                    <p className="text-sm text-muted-foreground">
                      {selectedFile.name} ({(selectedFile.size / 1024 / 1024).toFixed(2)} MB)
                    </p>
                  </div>
                ) : (
                  <div className="space-y-4">
                    <Image className="w-16 h-16 text-gray-400 mx-auto" />
                    <div>
                      <p className="text-lg font-medium">Selecione um arquivo</p>
                      <p className="text-sm text-muted-foreground">
                        Arraste e solte ou clique para selecionar
                      </p>
                      <p className="text-xs text-muted-foreground mt-1">
                        JPG, PNG, WebP ou PDF até 10MB
                      </p>
                    </div>
                  </div>
                )}
                
                <Input
                  ref={fileInputRef}
                  id="image-upload"
                  type="file"
                  accept="image/*,.pdf"
                  onChange={handleFileSelect}
                  className="hidden"
                />
                
                <div className="flex gap-2 justify-center mt-4">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => fileInputRef.current?.click()}
                  >
                    <Upload className="w-4 h-4 mr-2" />
                    {selectedFile ? 'Trocar Arquivo' : 'Selecionar Arquivo'}
                  </Button>
                  
                  {selectedFile && (
                    <Button
                      type="button"
                      variant="outline"
                      onClick={handleReset}
                    >
                      Remover
                    </Button>
                  )}
                </div>
              </div>
            </div>
          </div>

          {/* Processing Progress */}
          {isProcessing && (
            <Card>
              <CardContent className="p-6">
                <div className="flex items-center gap-3">
                  <Loader2 className="w-6 h-6 animate-spin text-blue-600" />
                  <div className="flex-1">
                    <p className="font-medium">Analisando imagem com IA...</p>
                    <p className="text-sm text-muted-foreground">
                      Extraindo informações dos produtos do cardápio. Isso pode levar alguns minutos.
                    </p>
                  </div>
                </div>
                <Progress value={undefined} className="mt-4" />
              </CardContent>
            </Card>
          )}

          {/* Import Results */}
          {importResult && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  Resultado da Importação
                  {importResult.success ? (
                    <CheckCircle className="w-5 h-5 text-green-600" />
                  ) : (
                    <XCircle className="w-5 h-5 text-red-600" />
                  )}
                </CardTitle>
                <CardDescription>
                  {importResult.success 
                    ? `${importResult.created_count || 0} produtos criados com sucesso`
                    : 'Houve problemas durante a importação'
                  }
                </CardDescription>
              </CardHeader>
              <CardContent>
                {/* Summary */}
                <div className="grid grid-cols-3 gap-4 mb-6">
                  <div className="text-center p-3 bg-green-50 rounded-lg">
                    <p className="text-2xl font-bold text-green-600">{importResult.created_count || 0}</p>
                    <p className="text-sm text-green-700">Criados</p>
                  </div>
                  <div className="text-center p-3 bg-red-50 rounded-lg">
                    <p className="text-2xl font-bold text-red-600">{importResult.errors?.length || 0}</p>
                    <p className="text-sm text-red-700">Erros</p>
                  </div>
                  <div className="text-center p-3 bg-blue-50 rounded-lg">
                    <p className="text-2xl font-bold text-blue-600">{importResult.credits_used || 0}</p>
                    <p className="text-sm text-blue-700">Créditos Usados</p>
                  </div>
                </div>

                {/* Products Table */}
                {importResult.products && importResult.products.length > 0 && (
                  <div className="space-y-4">
                    <h4 className="font-medium">Produtos Detectados</h4>
                    <div className="border rounded-lg overflow-hidden">
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>Status</TableHead>
                            <TableHead>Nome</TableHead>
                            <TableHead>Preço</TableHead>
                            <TableHead>Tags</TableHead>
                            <TableHead>Descrição</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {importResult.products.map((product, index) => (
                            <TableRow key={index}>
                              <TableCell>
                                {getStatusBadge(product.status || 'success')}
                              </TableCell>
                              <TableCell className="font-medium">{product.name}</TableCell>
                              <TableCell>R$ {parseFloat(product.price || '0').toFixed(2)}</TableCell>
                              <TableCell>
                                <div className="flex flex-wrap gap-1">
                                  {product.tags?.split(',').slice(0, 3).map((tag, tagIndex) => (
                                    <Badge key={tagIndex} variant="secondary" className="text-xs">
                                      {tag.trim()}
                                    </Badge>
                                  )) || []}
                                </div>
                              </TableCell>
                              <TableCell className="max-w-xs truncate">
                                {product.description}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </div>
                  </div>
                )}

                {/* Errors */}
                {importResult.errors && importResult.errors.length > 0 && (
                  <div className="space-y-4">
                    <h4 className="font-medium text-red-600">Erros Encontrados</h4>
                    <div className="space-y-2">
                      {importResult.errors.map((error, index) => (
                        <div key={index} className="flex items-center gap-2 p-3 bg-red-50 rounded-lg">
                          <XCircle className="w-4 h-4 text-red-600 flex-shrink-0" />
                          <span className="text-sm text-red-700">{error}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          )}

          {/* Actions */}
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={handleClose}>
              Cancelar
            </Button>
            <Button 
              onClick={handleImport} 
              disabled={!selectedFile || isProcessing}
              className="bg-gradient-primary"
            >
              {isProcessing ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Processando...
                </>
              ) : (
                <>
                  <Image className="w-4 h-4 mr-2" />
                  Analisar e Importar (1000 créditos)
                </>
              )}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
