import { useState, useEffect } from "react";
import { AlertTriangle, Clock, CheckCircle, XCircle, Filter } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useToast } from "@/hooks/use-toast";
import { apiClient } from "@/lib/api";

interface ErrorLog {
  id: string;
  tenant_id: string;
  customer_phone: string;
  customer_id?: string;
  user_message: string;
  tool_name: string;
  tool_args: string;
  error_message: string;
  error_type: string;
  user_response: string;
  severity: 'error' | 'warning' | 'critical';
  resolved: boolean;
  resolved_at?: string;
  resolved_by?: string;
  stack_trace?: string;
  created_at: string;
  updated_at: string;
}

interface ErrorStats {
  total_errors: number;
  unresolved_errors: number;
  critical_errors: number;
  errors_by_type: Array<{
    type: string;
    count: number;
  }>;
}

function ErrorLogs() {
  const [errorLogs, setErrorLogs] = useState<ErrorLog[]>([]);
  const [stats, setStats] = useState<ErrorStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [selectedError, setSelectedError] = useState<ErrorLog | null>(null);
  const [detailsOpen, setDetailsOpen] = useState(false);
  
  // Filters
  const [searchTerm, setSearchTerm] = useState("");
  const [severityFilter, setSeverityFilter] = useState("all");
  const [typeFilter, setTypeFilter] = useState("all");
  const [resolvedFilter, setResolvedFilter] = useState("all");
  
  // Pagination
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const itemsPerPage = 20;

  const { toast } = useToast();

  const fetchErrorLogs = async () => {
    try {
      setLoading(true);
      
      const params: any = {
        page: currentPage,
        limit: itemsPerPage,
      };
      
      if (severityFilter !== "all") params.severity = severityFilter;
      if (typeFilter !== "all") params.error_type = typeFilter;
      if (resolvedFilter !== "all") params.resolved = resolvedFilter;

      const response = await apiClient.getErrorLogs(params);
      
      if (response) {
        setErrorLogs(response.error_logs || []);
        if (response.pagination) {
          setTotalPages(response.pagination.pages);
        }
      }
    } catch (error) {
      console.error("Erro ao carregar logs:", error);
      toast({
        title: "Erro",
        description: "Não foi possível carregar os logs de erro",
        variant: "destructive",
      });
    } finally {
      setLoading(false);
    }
  };

  const fetchStats = async () => {
    try {
      const response = await apiClient.getErrorStats();
      setStats(response);
    } catch (error) {
      console.error("Erro ao carregar estatísticas:", error);
    }
  };

  const markAsResolved = async (errorId: string) => {
    try {
      await apiClient.resolveError(errorId);
      toast({
        title: "Sucesso",
        description: "Erro marcado como resolvido",
      });
      fetchErrorLogs();
      fetchStats();
    } catch (error) {
      console.error("Erro ao resolver:", error);
      toast({
        title: "Erro",
        description: "Não foi possível marcar como resolvido",
        variant: "destructive",
      });
    }
  };

  useEffect(() => {
    fetchErrorLogs();
    fetchStats();
  }, [currentPage, severityFilter, typeFilter, resolvedFilter]);

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical': return 'destructive';
      case 'warning': return 'secondary';
      default: return 'outline';
    }
  };

  const getSeverityIcon = (severity: string) => {
    switch (severity) {
      case 'critical': return <XCircle className="w-4 h-4" />;
      case 'warning': return <AlertTriangle className="w-4 h-4" />;
      default: return <AlertTriangle className="w-4 h-4" />;
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString('pt-BR');
  };

  const filteredLogs = errorLogs.filter(log => 
    searchTerm === "" || 
    log.customer_phone.includes(searchTerm) ||
    log.error_message.toLowerCase().includes(searchTerm.toLowerCase()) ||
    log.tool_name.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Logs de Erro</h1>
          <p className="text-muted-foreground">
            Monitore e gerencie erros do sistema AI
          </p>
        </div>
      </div>

      {/* Estatísticas */}
      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total de Erros</CardTitle>
              <AlertTriangle className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats.total_errors}</div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Não Resolvidos</CardTitle>
              <XCircle className="h-4 w-4 text-destructive" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-destructive">{stats.unresolved_errors}</div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Críticos</CardTitle>
              <AlertTriangle className="h-4 w-4 text-destructive" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-destructive">{stats.critical_errors}</div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Tipos</CardTitle>
              <Filter className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats.errors_by_type.length}</div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Filtros */}
      <Card>
        <CardHeader>
          <CardTitle>Filtros</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col md:flex-row gap-4">
            <div className="flex-1">
              <Input
                placeholder="Buscar por telefone, mensagem ou ferramenta..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full"
              />
            </div>
            
            <Select value={severityFilter} onValueChange={setSeverityFilter}>
              <SelectTrigger className="w-full md:w-40">
                <SelectValue placeholder="Severidade" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Todas</SelectItem>
                <SelectItem value="critical">Crítico</SelectItem>
                <SelectItem value="error">Erro</SelectItem>
                <SelectItem value="warning">Aviso</SelectItem>
              </SelectContent>
            </Select>

            <Select value={typeFilter} onValueChange={setTypeFilter}>
              <SelectTrigger className="w-full md:w-40">
                <SelectValue placeholder="Tipo" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Todos</SelectItem>
                <SelectItem value="db">Database</SelectItem>
                <SelectItem value="api">API</SelectItem>
                <SelectItem value="validation">Validação</SelectItem>
                <SelectItem value="ai">AI</SelectItem>
              </SelectContent>
            </Select>

            <Select value={resolvedFilter} onValueChange={setResolvedFilter}>
              <SelectTrigger className="w-full md:w-40">
                <SelectValue placeholder="Status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Todos</SelectItem>
                <SelectItem value="true">Resolvidos</SelectItem>
                <SelectItem value="false">Pendentes</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      {/* Tabela de Logs */}
      <Card>
        <CardHeader>
          <CardTitle>Logs de Erro</CardTitle>
          <CardDescription>
            {filteredLogs.length} registros encontrados
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Data</TableHead>
                  <TableHead>Cliente</TableHead>
                  <TableHead>Ferramenta</TableHead>
                  <TableHead>Tipo</TableHead>
                  <TableHead>Severidade</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Ações</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={7} className="text-center py-8">
                      Carregando...
                    </TableCell>
                  </TableRow>
                ) : filteredLogs.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} className="text-center py-8">
                      Nenhum erro encontrado
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredLogs.map((log) => (
                    <TableRow key={log.id}>
                      <TableCell className="font-mono text-sm">
                        {formatDate(log.created_at)}
                      </TableCell>
                      <TableCell>{log.customer_phone}</TableCell>
                      <TableCell>{log.tool_name}</TableCell>
                      <TableCell>
                        <Badge variant="outline">{log.error_type}</Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant={getSeverityColor(log.severity)}>
                          {getSeverityIcon(log.severity)}
                          <span className="ml-1">{log.severity}</span>
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {log.resolved ? (
                          <Badge variant="default">
                            <CheckCircle className="w-3 h-3 mr-1" />
                            Resolvido
                          </Badge>
                        ) : (
                          <Badge variant="secondary">
                            <Clock className="w-3 h-3 mr-1" />
                            Pendente
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell>
                        <div className="flex gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => {
                              setSelectedError(log);
                              setDetailsOpen(true);
                            }}
                          >
                            Ver Detalhes
                          </Button>
                          {!log.resolved && (
                            <Button
                              variant="default"
                              size="sm"
                              onClick={() => markAsResolved(log.id)}
                            >
                              Resolver
                            </Button>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>

          {/* Paginação */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between mt-4">
              <p className="text-sm text-muted-foreground">
                Página {currentPage} de {totalPages}
              </p>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setCurrentPage(prev => Math.max(1, prev - 1))}
                  disabled={currentPage === 1}
                >
                  Anterior
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setCurrentPage(prev => Math.min(totalPages, prev + 1))}
                  disabled={currentPage === totalPages}
                >
                  Próxima
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Dialog de Detalhes */}
      <Dialog open={detailsOpen} onOpenChange={setDetailsOpen}>
        <DialogContent className="max-w-4xl max-h-[80vh]">
          <DialogHeader>
            <DialogTitle>Detalhes do Erro</DialogTitle>
          </DialogHeader>
          {selectedError && (
            <ScrollArea className="max-h-[60vh] pr-4">
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="text-sm font-medium">Data:</label>
                    <p className="text-sm">{formatDate(selectedError.created_at)}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium">Cliente:</label>
                    <p className="text-sm">{selectedError.customer_phone}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium">Ferramenta:</label>
                    <p className="text-sm">{selectedError.tool_name}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium">Tipo:</label>
                    <p className="text-sm">{selectedError.error_type}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium">Severidade:</label>
                    <Badge variant={getSeverityColor(selectedError.severity)}>
                      {selectedError.severity}
                    </Badge>
                  </div>
                  <div>
                    <label className="text-sm font-medium">Status:</label>
                    {selectedError.resolved ? (
                      <Badge variant="default">Resolvido</Badge>
                    ) : (
                      <Badge variant="secondary">Pendente</Badge>
                    )}
                  </div>
                </div>

                <div>
                  <label className="text-sm font-medium">Mensagem do Usuário:</label>
                  <p className="text-sm bg-muted p-3 rounded mt-1">{selectedError.user_message}</p>
                </div>

                <div>
                  <label className="text-sm font-medium">Erro Técnico:</label>
                  <p className="text-sm bg-destructive/10 p-3 rounded mt-1 font-mono">
                    {selectedError.error_message}
                  </p>
                </div>

                <div>
                  <label className="text-sm font-medium">Resposta ao Usuário:</label>
                  <p className="text-sm bg-primary/10 p-3 rounded mt-1">{selectedError.user_response}</p>
                </div>

                {selectedError.tool_args && (
                  <div>
                    <label className="text-sm font-medium">Argumentos da Ferramenta:</label>
                    <pre className="text-xs bg-muted p-3 rounded mt-1 overflow-x-auto">
                      {selectedError.tool_args}
                    </pre>
                  </div>
                )}

                {selectedError.stack_trace && (
                  <div>
                    <label className="text-sm font-medium">Stack Trace:</label>
                    <pre className="text-xs bg-muted p-3 rounded mt-1 overflow-x-auto">
                      {selectedError.stack_trace}
                    </pre>
                  </div>
                )}

                {selectedError.resolved && (
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="text-sm font-medium">Resolvido em:</label>
                      <p className="text-sm">{selectedError.resolved_at ? formatDate(selectedError.resolved_at) : 'N/A'}</p>
                    </div>
                    <div>
                      <label className="text-sm font-medium">Resolvido por:</label>
                      <p className="text-sm">{selectedError.resolved_by || 'N/A'}</p>
                    </div>
                  </div>
                )}
              </div>
            </ScrollArea>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

export default ErrorLogs;
