import { useState, useEffect } from "react";
import { Upload, Download, FileText, CheckCircle, XCircle, Clock, AlertCircle } from "lucide-react";
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
import { useCreateProductImportJob, useImportJobProgress, useDownloadImportTemplate, queryKeys } from "@/lib/api/hooks";
import { toast } from "sonner";
import { useQueryClient } from "@tanstack/react-query";

interface ProductAsyncImportDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ProductAsyncImportDialog({ open, onOpenChange }: ProductAsyncImportDialogProps) {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [jobId, setJobId] = useState<string | null>(null);
  const [isStarting, setIsStarting] = useState(false);

  const queryClient = useQueryClient();
  const createImportJob = useCreateProductImportJob();
  const downloadTemplate = useDownloadImportTemplate();
  
  // Poll for progress when job is created
  const { data: progressData, isLoading: isLoadingProgress } = useImportJobProgress(
    jobId, 
    !!jobId
  );

  // Reset state when dialog closes
  useEffect(() => {
    if (!open) {
      setSelectedFile(null);
      setJobId(null);
      setIsStarting(false);
    }
  }, [open]);

  // Show success notification when job completes (without auto-closing)
  useEffect(() => {
    if (progressData?.job?.status === 'completed' && jobId) {
      toast.success(`Importação concluída! ${progressData.job.successful_items} produtos processados com sucesso.`);
    }
    if (progressData?.job?.status === 'failed' && jobId) {
      toast.error(`Importação falhou! ${progressData.job.failed_items} produtos falharam.`);
    }
  }, [progressData?.job?.status, jobId]);

  const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file && (file.type === 'text/csv' || file.name.endsWith('.csv'))) {
      setSelectedFile(file);
      setJobId(null); // Reset job if file changes
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

  const handleStartImport = async () => {
    if (!selectedFile) return;

    setIsStarting(true);
    try {
      const formData = new FormData();
      formData.append('file', selectedFile);

      const response = await createImportJob.mutateAsync(formData);
      setJobId(response.job_id);
      toast.success('Importação iniciada! Acompanhe o progresso abaixo.');
    } catch (error: any) {
      toast.error(error.message || 'Erro ao iniciar a importação');
    } finally {
      setIsStarting(false);
    }
  };

  const handleClose = () => {
    // If job completed successfully, force refresh the products list
    if (progressData?.job?.status === 'completed' && progressData?.job?.successful_items > 0) {
      queryClient.invalidateQueries({ queryKey: queryKeys.products });
    }
    
    onOpenChange(false);
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'pending':
        return <Clock className="w-4 h-4 text-yellow-500" />;
      case 'processing':
        return <Clock className="w-4 h-4 text-blue-500 animate-spin" />;
      case 'completed':
        return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'failed':
        return <XCircle className="w-4 h-4 text-red-500" />;
      default:
        return <AlertCircle className="w-4 h-4 text-gray-500" />;
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'pending':
        return <Badge variant="outline" className="bg-yellow-50 text-yellow-700 border-yellow-200">Pendente</Badge>;
      case 'processing':
        return <Badge variant="outline" className="bg-blue-50 text-blue-700 border-blue-200">Processando</Badge>;
      case 'completed':
        return <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200">Concluído</Badge>;
      case 'failed':
        return <Badge variant="destructive">Falhou</Badge>;
      default:
        return <Badge variant="secondary">{status}</Badge>;
    }
  };

  const isJobInProgress = progressData?.job?.status === 'pending' || progressData?.job?.status === 'processing';
  const isJobCompleted = progressData?.job?.status === 'completed';
  const isJobFailed = progressData?.job?.status === 'failed';

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Importar Produtos (Assíncrono)</DialogTitle>
          <DialogDescription>
            Importe produtos em massa através de um arquivo CSV. O processo é executado em segundo plano.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {/* Template Download */}
          {!jobId && (
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
          )}

          {/* File Upload Section */}
          {!jobId && (
            <>
              <div className="space-y-2">
                <Label htmlFor="csv-file">Arquivo CSV</Label>
                <Input
                  id="csv-file"
                  type="file"
                  accept=".csv"
                  onChange={handleFileSelect}
                  disabled={isJobInProgress || isStarting}
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
                  onClick={handleStartImport}
                  disabled={!selectedFile || isJobInProgress || isStarting}
                  className="flex-1"
                >
                  {isStarting ? (
                    <>
                      <Clock className="w-4 h-4 mr-2 animate-spin" />
                      Iniciando...
                    </>
                  ) : (
                    <>
                      <Upload className="w-4 h-4 mr-2" />
                      Iniciar Importação
                    </>
                  )}
                </Button>
                <Button variant="outline" onClick={handleClose}>
                  Cancelar
                </Button>
              </div>
            </>
          )}

          {/* Close Button for Active Jobs */}
          {jobId && (
            <div className="flex justify-end gap-2">
              {isJobCompleted && (
                <div className="flex items-center text-sm text-green-600 mr-4">
                  <CheckCircle className="w-4 h-4 mr-2" />
                  Importação finalizada! Você pode fechar este modal.
                </div>
              )}
              {isJobFailed && (
                <div className="flex items-center text-sm text-red-600 mr-4">
                  <XCircle className="w-4 h-4 mr-2" />
                  Importação falhou. Verifique os detalhes abaixo.
                </div>
              )}
              <Button 
                variant={isJobCompleted ? "default" : "outline"} 
                onClick={handleClose}
                className={isJobCompleted ? "bg-green-600 hover:bg-green-700" : ""}
              >
                {isJobCompleted ? 'Fechar e Finalizar' : 
                 isJobFailed ? 'Fechar' : 
                 'Fechar e Continuar em Segundo Plano'}
              </Button>
            </div>
          )}

          {/* Job Progress */}
          {jobId && progressData && (
            <div className="space-y-4">
              <div className="border rounded-lg p-4">
                <div className="flex items-center justify-between mb-3">
                  <h4 className="font-medium">Status da Importação</h4>
                  <div className="flex items-center gap-2">
                    {getStatusIcon(progressData.job.status)}
                    {getStatusBadge(progressData.job.status)}
                  </div>
                </div>

                {/* Progress Bar */}
                <div className="space-y-2">
                  <div className="flex justify-between text-sm">
                    <span>Progresso</span>
                    <span>{Math.round(progressData.progress_percentage)}%</span>
                  </div>
                  <Progress value={progressData.progress_percentage} className="h-2" />
                  <div className="flex justify-between text-xs text-muted-foreground">
                    <span>
                      {progressData.job.processed_items} de {progressData.job.total_items} processados
                    </span>
                    <span>
                      ID: {jobId}
                    </span>
                  </div>
                </div>

                {/* Summary Stats */}
                <div className="grid grid-cols-4 gap-4 mt-4 text-sm">
                  <div className="text-center">
                    <div className="text-2xl font-bold text-blue-600">
                      {progressData.job.total_items}
                    </div>
                    <div className="text-muted-foreground">Total</div>
                  </div>
                  <div className="text-center">
                    <div className="text-2xl font-bold text-green-600">
                      {progressData.job.successful_items}
                    </div>
                    <div className="text-muted-foreground">Sucesso</div>
                  </div>
                  <div className="text-center">
                    <div className="text-2xl font-bold text-yellow-600">
                      {progressData.job.processed_items}
                    </div>
                    <div className="text-muted-foreground">Processados</div>
                  </div>
                  <div className="text-center">
                    <div className="text-2xl font-bold text-red-600">
                      {progressData.job.failed_items}
                    </div>
                    <div className="text-muted-foreground">Erros</div>
                  </div>
                </div>

                {/* Final Summary for Completed Jobs */}
                {isJobCompleted && (
                  <div className="mt-4 p-4 bg-green-50 border border-green-200 rounded-lg">
                    <div className="flex items-center gap-2 text-green-800 mb-2">
                      <CheckCircle className="w-5 h-5" />
                      <span className="font-semibold">Importação Concluída com Sucesso!</span>
                    </div>
                    <div className="text-sm text-green-700">
                      <p className="mb-1">
                        <strong>{progressData.job.successful_items}</strong> produtos foram processados com sucesso
                        {progressData.job.failed_items > 0 && 
                          ` e ${progressData.job.failed_items} apresentaram erros`
                        }.
                      </p>
                      <p>Você pode fechar este modal quando desejar. Os produtos já estão disponíveis em seu sistema.</p>
                    </div>
                  </div>
                )}

                {isJobFailed && (
                  <div className="mt-4 p-4 bg-red-50 border border-red-200 rounded-lg">
                    <div className="flex items-center gap-2 text-red-800 mb-2">
                      <XCircle className="w-5 h-5" />
                      <span className="font-semibold">Importação Falhou</span>
                    </div>
                    <div className="text-sm text-red-700">
                      <p>Verifique os detalhes do erro abaixo e tente novamente com um arquivo corrigido.</p>
                    </div>
                  </div>
                )}

                {/* Error Message */}
                {progressData.job.error_message && (
                  <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded-lg">
                    <div className="flex items-center gap-2 text-red-800">
                      <XCircle className="w-4 h-4" />
                      <span className="font-medium">Erro</span>
                    </div>
                    <p className="text-sm text-red-700 mt-1">
                      {progressData.job.error_message}
                    </p>
                  </div>
                )}
              </div>

              {/* Detailed Results (when completed) */}
              {isJobCompleted && progressData.details && (
                <div className="border rounded-lg">
                  <div className="p-4 border-b">
                    <h4 className="font-medium">Resultados Detalhados</h4>
                  </div>
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
                      {progressData.details.results.map((result) => (
                        <TableRow key={result.row_number}>
                          <TableCell>{result.row_number}</TableCell>
                          <TableCell className="font-medium">{result.name}</TableCell>
                          <TableCell>{result.sku || '-'}</TableCell>
                          <TableCell>
                            <div className="flex items-center gap-2">
                              {result.status === 'created' || result.status === 'updated' ? (
                                <CheckCircle className="w-4 h-4 text-green-500" />
                              ) : (
                                <XCircle className="w-4 h-4 text-red-500" />
                              )}
                              <Badge variant={result.status === 'error' ? 'destructive' : 'default'}>
                                {result.status === 'created' ? 'Criado' :
                                 result.status === 'updated' ? 'Atualizado' : 'Erro'}
                              </Badge>
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
              )}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
