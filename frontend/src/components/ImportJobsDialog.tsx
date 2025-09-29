import { useState, useEffect } from "react";
import { 
  Dialog, 
  DialogContent, 
  DialogDescription, 
  DialogHeader, 
  DialogTitle 
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { RefreshCw, Clock, CheckCircle, XCircle, AlertCircle, FileText } from "lucide-react";
import { format } from "date-fns";
import { ptBR } from "date-fns/locale";
import { API_BASE_URL } from "@/lib/api/client";

interface ImportJob {
  id: string;
  created_at: string;
  updated_at: string;
  tenant_id: string;
  user_id: string;
  type: string;
  status: string;
  file_name: string;
  file_path: string;
  total_records: number;
  processed_records: number;
  success_records: number;
  error_records: number;
  started_at: string | null;
  completed_at: string | null;
}

interface ImportJobsResponse {
  jobs: ImportJob[];
  total: number;
  page: number;
  limit: number;
}

interface ImportJobsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ImportJobsDialog({ open, onOpenChange }: ImportJobsDialogProps) {
  const [jobs, setJobs] = useState<ImportJob[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [limit] = useState(10);

  const fetchJobs = async () => {
    const token = localStorage.getItem('access_token');
    if (!token) return;
    
    setLoading(true);
    try {
      const response = await fetch(
        `${API_BASE_URL}/import/jobs?page=${page}&limit=${limit}`,
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        }
      );

      if (response.ok) {
        const data: ImportJobsResponse = await response.json();
        setJobs(data.jobs);
        setTotal(data.total);
      }
    } catch (error) {
      console.error("Erro ao buscar jobs:", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (open) {
      fetchJobs();
    }
  }, [open, page]);

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "completed":
        return (
          <Badge variant="secondary" className="bg-green-100 text-green-800">
            <CheckCircle className="w-3 h-3 mr-1" />
            Concluído
          </Badge>
        );
      case "processing":
        return (
          <Badge variant="secondary" className="bg-blue-100 text-blue-800">
            <Clock className="w-3 h-3 mr-1" />
            Processando
          </Badge>
        );
      case "failed":
        return (
          <Badge variant="destructive">
            <XCircle className="w-3 h-3 mr-1" />
            Falhou
          </Badge>
        );
      case "pending":
        return (
          <Badge variant="outline">
            <AlertCircle className="w-3 h-3 mr-1" />
            Pendente
          </Badge>
        );
      default:
        return <Badge variant="outline">{status}</Badge>;
    }
  };

  const getProgress = (job: ImportJob) => {
    if (job.total_records === 0) return 0;
    return Math.round((job.processed_records / job.total_records) * 100);
  };

  const formatDuration = (started: string | null, completed: string | null) => {
    if (!started) return "-";
    
    const startTime = new Date(started);
    const endTime = completed ? new Date(completed) : new Date();
    const duration = endTime.getTime() - startTime.getTime();
    
    if (duration < 60000) {
      return `${Math.round(duration / 1000)}s`;
    } else if (duration < 3600000) {
      return `${Math.round(duration / 60000)}m`;
    } else {
      return `${Math.round(duration / 3600000)}h`;
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-6xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <div className="flex items-center justify-between">
            <div>
              <DialogTitle className="flex items-center gap-2">
                <FileText className="w-5 h-5" />
                Jobs de Importação
              </DialogTitle>
              <DialogDescription>
                Acompanhe o progresso das suas importações de produtos
              </DialogDescription>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => fetchJobs()}
              disabled={loading}
            >
              <RefreshCw className={`w-4 h-4 mr-2 ${loading ? "animate-spin" : ""}`} />
              Atualizar
            </Button>
          </div>
        </DialogHeader>

        <div className="mt-6">
          {loading ? (
            <div className="space-y-4">
              {Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="flex items-center space-x-4">
                  <Skeleton className="h-4 w-[100px]" />
                  <Skeleton className="h-4 w-[200px]" />
                  <Skeleton className="h-4 w-[100px]" />
                  <Skeleton className="h-4 w-[150px]" />
                </div>
              ))}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Arquivo</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Progresso</TableHead>
                  <TableHead>Registros</TableHead>
                  <TableHead>Duração</TableHead>
                  <TableHead>Criado em</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {jobs.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                      Nenhum job de importação encontrado
                    </TableCell>
                  </TableRow>
                ) : (
                  jobs.map((job) => (
                    <TableRow key={job.id}>
                      <TableCell className="font-medium">
                        <div className="max-w-[200px] truncate" title={job.file_name}>
                          {job.file_name}
                        </div>
                      </TableCell>
                      <TableCell>{getStatusBadge(job.status)}</TableCell>
                      <TableCell>
                        <div className="space-y-1">
                          <div className="text-sm text-muted-foreground">
                            {getProgress(job)}%
                          </div>
                          <div className="w-full bg-gray-200 rounded-full h-2">
                            <div
                              className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                              style={{ width: `${getProgress(job)}%` }}
                            />
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="text-sm space-y-1">
                          <div>
                            <span className="text-green-600">{job.success_records}</span>
                            {job.error_records > 0 && (
                              <span className="text-red-600"> / {job.error_records} erros</span>
                            )}
                          </div>
                          <div className="text-muted-foreground">
                            de {job.total_records} total
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        {formatDuration(job.started_at, job.completed_at)}
                      </TableCell>
                      <TableCell>
                        {format(new Date(job.created_at), "dd/MM/yyyy HH:mm", {
                          locale: ptBR,
                        })}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          )}
        </div>

        {total > limit && (
          <div className="flex items-center justify-between mt-4">
            <div className="text-sm text-muted-foreground">
              Mostrando {Math.min(page * limit, total)} de {total} jobs
            </div>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page === 1}
                onClick={() => setPage(page - 1)}
              >
                Anterior
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={page * limit >= total}
                onClick={() => setPage(page + 1)}
              >
                Próximo
              </Button>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
