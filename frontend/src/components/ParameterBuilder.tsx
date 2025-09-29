import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Plus, Trash2, Eye, EyeOff, Code } from "lucide-react";

export interface FunctionParameter {
  id: string;
  name: string;
  type: "string" | "number" | "boolean" | "array";
  description: string;
  required: boolean;
  defaultValue?: string;
}

interface ParameterBuilderProps {
  value: string; // JSON schema atual
  onChange: (jsonSchema: string) => void;
  className?: string;
}

export function ParameterBuilder({ value, onChange, className }: ParameterBuilderProps) {
  const [parameters, setParameters] = useState<FunctionParameter[]>([]);
  const [showJsonPreview, setShowJsonPreview] = useState(false);
  const [jsonError, setJsonError] = useState<string>("");

  // Inicializar parâmetros a partir do JSON schema existente
  useEffect(() => {
    if (value && value !== "{}") {
      try {
        const schema = JSON.parse(value);
        if (schema.type === "object" && schema.properties) {
          const loadedParams: FunctionParameter[] = [];
          Object.entries(schema.properties).forEach(([key, prop]: [string, any]) => {
            loadedParams.push({
              id: Math.random().toString(36).substr(2, 9),
              name: key,
              type: prop.type || "string",
              description: prop.description || "",
              required: schema.required?.includes(key) || false,
              defaultValue: prop.default?.toString() || ""
            });
          });
          setParameters(loadedParams);
        }
      } catch (error) {
        setJsonError("JSON inválido");
      }
    }
  }, []);

  // Gerar JSON schema quando parâmetros mudarem
  useEffect(() => {
    generateJsonSchema();
  }, [parameters]);

  const generateJsonSchema = () => {
    if (parameters.length === 0) {
      onChange("{}");
      setJsonError("");
      return;
    }

    try {
      const properties: any = {};
      const required: string[] = [];

      parameters.forEach(param => {
        if (!param.name.trim()) return;

        properties[param.name] = {
          type: param.type,
          ...(param.description && { description: param.description }),
          ...(param.defaultValue && { default: convertValue(param.defaultValue, param.type) })
        };

        if (param.required) {
          required.push(param.name);
        }
      });

      const schema = {
        type: "object",
        properties,
        ...(required.length > 0 && { required })
      };

      onChange(JSON.stringify(schema, null, 2));
      setJsonError("");
    } catch (error) {
      setJsonError("Erro ao gerar JSON schema");
    }
  };

  const convertValue = (value: string, type: string) => {
    switch (type) {
      case "number":
        return Number(value) || 0;
      case "boolean":
        return value.toLowerCase() === "true";
      case "array":
        try {
          return JSON.parse(value);
        } catch {
          return [];
        }
      default:
        return value;
    }
  };

  const addParameter = () => {
    const newParam: FunctionParameter = {
      id: Math.random().toString(36).substr(2, 9),
      name: "",
      type: "string",
      description: "",
      required: false
    };
    setParameters([...parameters, newParam]);
  };

  const updateParameter = (id: string, updates: Partial<FunctionParameter>) => {
    setParameters(params =>
      params.map(param =>
        param.id === id ? { ...param, ...updates } : param
      )
    );
  };

  const removeParameter = (id: string) => {
    setParameters(params => params.filter(param => param.id !== id));
  };

  const getTypeColor = (type: string) => {
    switch (type) {
      case "string": return "bg-blue-100 text-blue-800";
      case "number": return "bg-green-100 text-green-800";
      case "boolean": return "bg-purple-100 text-purple-800";
      case "array": return "bg-orange-100 text-orange-800";
      default: return "bg-gray-100 text-gray-800";
    }
  };

  const getTypeExample = (type: string) => {
    switch (type) {
      case "string": return "Ex: nome, email, mensagem";
      case "number": return "Ex: 10, 3.14, 100";
      case "boolean": return "true ou false";
      case "array": return '["item1", "item2"]';
      default: return "";
    }
  };

  return (
    <div className={`space-y-4 ${className}`}>
      <div className="flex items-center justify-between">
        <Label className="text-base font-medium">Parâmetros da Função</Label>
        <div className="flex items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => setShowJsonPreview(!showJsonPreview)}
          >
            {showJsonPreview ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            {showJsonPreview ? "Ocultar" : "Ver JSON"}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={addParameter}
          >
            <Plus className="h-4 w-4" />
            Adicionar Parâmetro
          </Button>
        </div>
      </div>

      {parameters.length === 0 ? (
        <Card className="border-dashed">
          <CardContent className="flex flex-col items-center justify-center py-8">
            <Code className="h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-muted-foreground text-center">
              Nenhum parâmetro configurado.<br />
              Clique em "Adicionar Parâmetro" para começar.
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {parameters.map((param, index) => (
            <Card key={param.id} className="border-l-4 border-l-blue-500">
              <CardHeader className="pb-3">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-sm flex items-center gap-2">
                    <span className="bg-blue-100 text-blue-800 px-2 py-1 rounded text-xs font-mono">
                      #{index + 1}
                    </span>
                    <Badge className={getTypeColor(param.type)}>
                      {param.type}
                    </Badge>
                    {param.required && (
                      <Badge variant="destructive" className="text-xs">
                        Obrigatório
                      </Badge>
                    )}
                  </CardTitle>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => removeParameter(param.id)}
                    className="text-red-600 hover:text-red-700"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label className="text-xs">Nome do Parâmetro *</Label>
                    <Input
                      value={param.name}
                      onChange={(e) => updateParameter(param.id, { name: e.target.value })}
                      placeholder="Ex: userId, email, quantidade"
                      className="text-sm"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label className="text-xs">Tipo</Label>
                    <Select
                      value={param.type}
                      onValueChange={(value: any) => updateParameter(param.id, { type: value })}
                    >
                      <SelectTrigger className="text-sm">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="string">
                          <div className="flex items-center gap-2">
                            <Badge className="bg-blue-100 text-blue-800 text-xs">string</Badge>
                            Texto
                          </div>
                        </SelectItem>
                        <SelectItem value="number">
                          <div className="flex items-center gap-2">
                            <Badge className="bg-green-100 text-green-800 text-xs">number</Badge>
                            Número
                          </div>
                        </SelectItem>
                        <SelectItem value="boolean">
                          <div className="flex items-center gap-2">
                            <Badge className="bg-purple-100 text-purple-800 text-xs">boolean</Badge>
                            Verdadeiro/Falso
                          </div>
                        </SelectItem>
                        <SelectItem value="array">
                          <div className="flex items-center gap-2">
                            <Badge className="bg-orange-100 text-orange-800 text-xs">array</Badge>
                            Lista
                          </div>
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <div className="space-y-2">
                  <Label className="text-xs">Descrição</Label>
                  <Textarea
                    value={param.description}
                    onChange={(e) => updateParameter(param.id, { description: e.target.value })}
                    placeholder="Descreva o que este parâmetro representa e como deve ser usado"
                    className="text-sm min-h-[60px]"
                  />
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label className="text-xs">Valor Padrão (opcional)</Label>
                    <Input
                      value={param.defaultValue || ""}
                      onChange={(e) => updateParameter(param.id, { defaultValue: e.target.value })}
                      placeholder={getTypeExample(param.type)}
                      className="text-sm"
                    />
                  </div>
                  <div className="flex items-center space-x-2 mt-6">
                    <Switch
                      checked={param.required}
                      onCheckedChange={(checked) => updateParameter(param.id, { required: checked })}
                    />
                    <Label className="text-xs">Parâmetro obrigatório</Label>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {showJsonPreview && (
        <Card className="bg-gray-50">
          <CardHeader className="pb-3">
            <CardTitle className="text-sm flex items-center gap-2">
              <Code className="h-4 w-4" />
              JSON Schema Gerado
            </CardTitle>
          </CardHeader>
          <CardContent>
            {jsonError ? (
              <div className="text-red-600 text-sm p-3 bg-red-50 rounded border">
                ❌ {jsonError}
              </div>
            ) : (
              <pre className="text-xs bg-white p-3 rounded border font-mono overflow-x-auto">
                {value || "{}"}
              </pre>
            )}
          </CardContent>
        </Card>
      )}

      {parameters.length > 0 && (
        <div className="bg-blue-50 p-4 rounded border-l-4 border-l-blue-500">
          <div className="flex items-start gap-3">
            <div className="bg-blue-500 rounded-full p-1">
              <Code className="h-4 w-4 text-white" />
            </div>
            <div className="flex-1">
              <h4 className="font-medium text-blue-900 mb-1">Como funciona</h4>
              <p className="text-sm text-blue-800">
                A IA receberá estes parâmetros quando chamar sua função. Configure nome, tipo e descrição 
                para que a IA entenda exatamente o que enviar.
              </p>
            </div>
          </div>
        </div>
      )}

      {parameters.length === 0 && (
        <div className="bg-gray-50 p-4 rounded border">
          <h4 className="font-medium text-gray-900 mb-3">💡 Exemplos de Parâmetros</h4>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
            <div className="space-y-2">
              <div className="font-medium text-gray-700">Para buscar usuário:</div>
              <div className="pl-3 border-l-2 border-blue-200">
                <div><strong>userId</strong> (string): ID do usuário a buscar</div>
                <div><strong>includeDetails</strong> (boolean): Incluir detalhes extras</div>
              </div>
            </div>
            <div className="space-y-2">
              <div className="font-medium text-gray-700">Para análise AML:</div>
              <div className="pl-3 border-l-2 border-green-200">
                <div><strong>moeda</strong> (string): Nome da criptomoeda</div>
                <div><strong>endereco</strong> (string): Endereço da carteira</div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}