import { useState, useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Checkbox } from "@/components/ui/checkbox";
import { PhoneNumberInput } from "@/components/ui/phone-input";
import { cleanPhoneForStorage } from "@/lib/phone-utils";
import { API_BASE_URL } from "@/lib/api/client";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useCreateTenant, useCreateTenantAdmin, usePlans } from "@/lib/api/hooks";
import { apiClient } from "@/lib/api";
import { toast } from "@/hooks/use-toast";
import { Loader2 } from "lucide-react";
import { Plan } from "@/lib/api/types";

const formSchema = z.object({
  name: z.string().min(2, "Nome deve ter pelo menos 2 caracteres"),
  domain: z.string().min(2, "Domínio deve ter pelo menos 2 caracteres").optional().or(z.literal("")),
  plan_id: z.string().min(1, "Selecione um plano"),
  status: z.enum(["active", "inactive", "suspended"]),
  business_type: z.enum(["sales"]),
  business_category: z.string().default("loja"),
  cost_per_message: z.number().min(0, "Custo por mensagem deve ser 0 ou maior").default(0),
  enable_ai_prompt_customization: z.boolean().default(false),
  is_public_store: z.boolean().default(false),
  tag: z.string().optional(),
  // max_users and max_messages removed - now come from plan
  // Admin user fields
  admin_name: z.string().min(1, "Nome é obrigatório"),
  admin_email: z.string().email("Email inválido"),
  admin_phone: z.string().min(1, "Telefone é obrigatório"),
  admin_password: z.string().optional(),
  send_credentials_by_email: z.boolean().default(false),
}).refine((data) => {
  // Se is_public_store for true, tag deve ser obrigatório e não vazio
  if (data.is_public_store && (!data.tag || data.tag.trim() === "")) {
    return false;
  }
  return true;
}, {
  message: "TAG é obrigatório para lojas públicas",
  path: ["tag"],
}).refine((data) => {
  // Se send_credentials_by_email for false, admin_password é obrigatório
  if (!data.send_credentials_by_email && (!data.admin_password || data.admin_password.length < 6)) {
    return false;
  }
  return true;
}, {
  message: "Senha deve ter pelo menos 6 caracteres quando não enviar por email",
  path: ["admin_password"],
});

type FormData = z.infer<typeof formSchema>;

interface CreateTenantDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

// Função para gerar senha aleatória de 8 dígitos (números + letras)
const generateRandomPassword = (): string => {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  let password = '';
  for (let i = 0; i < 8; i++) {
    password += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return password;
};

export function CreateTenantDialog({ open, onOpenChange }: CreateTenantDialogProps) {
  const [isLoading, setIsLoading] = useState(false);
  const createTenantMutation = useCreateTenant();
  const createTenantAdminMutation = useCreateTenantAdmin();
  const { data: plans = [], isLoading: loadingPlans } = usePlans();

  const form = useForm<FormData>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
      domain: "",
      plan_id: "",
      status: "active",
      business_type: "sales",
      cost_per_message: 0,
      enable_ai_prompt_customization: false,
      is_public_store: false,
      tag: "",
      business_category: "loja",
      admin_name: "",
      admin_email: "",
      admin_phone: "+55",
      admin_password: "",
      send_credentials_by_email: false,
    },
  });

  const onSubmit = async (data: FormData) => {
    setIsLoading(true);
    try {
      // Gerar senha automaticamente se envio por email estiver habilitado
      const finalPassword = data.send_credentials_by_email 
        ? generateRandomPassword() 
        : data.admin_password!;

      // First, create the tenant
      const tenant = await createTenantMutation.mutateAsync({
        name: data.name,
        domain: data.domain || undefined,
        plan_id: data.plan_id,
        status: data.status,
        business_type: data.business_type,
        business_category: data.business_category,
        cost_per_message: data.cost_per_message,
        enable_ai_prompt_customization: data.enable_ai_prompt_customization,
        is_public_store: data.is_public_store,
        tag: data.tag || undefined,
        about: "",
        // max_users and max_messages now come from plan relationship
      });

      // Then, create the tenant admin
      await createTenantAdminMutation.mutateAsync({
        tenant_id: tenant.id,
        name: data.admin_name,
        email: data.admin_email,
        phone: cleanPhoneForStorage(data.admin_phone || ""),
        password: finalPassword,
      });

      // Enviar email com credenciais se solicitado
      if (data.send_credentials_by_email) {
        try {
          await fetch(`${API_BASE_URL}/admin/send-tenant-credentials`, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'Authorization': `Bearer ${localStorage.getItem('access_token')}`,
            },
            body: JSON.stringify({
              tenant_id: tenant.id,
              tenant_name: data.name,
              admin_name: data.admin_name,
              admin_email: data.admin_email,
              admin_password: finalPassword,
            }),
          });
          
          toast({
            title: "Sucesso",
            description: "Empresa criada e credenciais enviadas por email!",
          });
        } catch (emailError) {
          console.error('Erro ao enviar email:', emailError);
          toast({
            title: "Empresa criada com sucesso",
            description: `Credenciais: Email: ${data.admin_email}, Senha: ${finalPassword}`,
            variant: "default",
          });
        }
      } else {
        toast({
          title: "Sucesso",
          description: "Empresa e administrador criados com sucesso!",
        });
      }
      
      form.reset();
      onOpenChange(false);
    } catch (error: any) {
      toast({
        title: "Erro",
        description: error.message || "Erro ao criar empresa",
        variant: "destructive",
      });
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-4xl w-full max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Nova Empresa</DialogTitle>
          <DialogDescription>
            Crie uma nova empresa no sistema
          </DialogDescription>
        </DialogHeader>
        
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
            {/* Seção: Informações da Empresa */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-gray-900 border-b pb-2">
                Informações da Empresa
              </h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <FormField
                  control={form.control}
                  name="name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Nome da Empresa</FormLabel>
                      <FormControl>
                        <Input placeholder="Ex: Minha Empresa Ltda" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="domain"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Domínio (opcional)</FormLabel>
                      <FormControl>
                        <Input placeholder="Ex: minhaempresa" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="plan_id"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Plano</FormLabel>
                      <Select onValueChange={field.onChange} defaultValue={field.value || ""}>
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue placeholder="Selecione um plano" />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          {loadingPlans ? (
                            <SelectItem value="loading" disabled>Carregando...</SelectItem>
                          ) : (
                            plans
                              .sort((a, b) => a.price - b.price)
                              .map((plan) => (
                                <SelectItem key={plan.id} value={plan.id}>
                                  {plan.name} - R$ {plan.price.toFixed(2)}
                                </SelectItem>
                              ))
                          )}
                        </SelectContent>
                      </Select>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="status"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Status</FormLabel>
                      <Select onValueChange={field.onChange} defaultValue={field.value}>
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue placeholder="Selecione um status" />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          <SelectItem value="active">Ativo</SelectItem>
                          <SelectItem value="inactive">Inativo</SelectItem>
                          <SelectItem value="suspended">Suspenso</SelectItem>
                        </SelectContent>
                      </Select>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="business_type"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Tipo de Negócio</FormLabel>
                      <Select onValueChange={field.onChange} defaultValue={field.value}>
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue placeholder="Selecione o tipo de negócio" />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          <SelectItem value="sales">Vendas</SelectItem>
                        </SelectContent>
                      </Select>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="business_category"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Categoria do Negócio</FormLabel>
                      <Select onValueChange={field.onChange} defaultValue={field.value}>
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue placeholder="Selecione a categoria do negócio" />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          <SelectItem value="loja">Loja</SelectItem>
                          <SelectItem value="farmacia">Farmácia</SelectItem>
                          <SelectItem value="hamburgeria">Hambúrgeria</SelectItem>
                          <SelectItem value="pizzaria">Pizzaria</SelectItem>
                          <SelectItem value="acaiteria">Açaiteria</SelectItem>
                          <SelectItem value="restaurante">Restaurante</SelectItem>
                        </SelectContent>
                      </Select>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>

              {/* Campo de custo por mensagem - fica sozinho por ser específico */}
              <FormField
                control={form.control}
                name="cost_per_message"
                render={({ field }) => (
                  <FormItem className="max-w-md">
                    <FormLabel>Custo por Mensagem IA (créditos)</FormLabel>
                    <FormControl>
                      <Input 
                        placeholder="0"
                        type="number"
                        min="0"
                        {...field}
                        onChange={e => field.onChange(parseInt(e.target.value) || 0)}
                      />
                    </FormControl>
                    <p className="text-sm text-muted-foreground">
                      0 = grátis, {'>'}0 = créditos descontados por mensagem processada pela IA
                    </p>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* TAG da loja condicional */}
              {form.watch("is_public_store") && (
                <FormField
                  control={form.control}
                  name="tag"
                  render={({ field }) => (
                    <FormItem className="max-w-md">
                      <FormLabel>TAG da Loja *</FormLabel>
                      <FormControl>
                        <Input {...field} placeholder="Ex: farmacia-central, pizzaria-dello" />
                      </FormControl>
                      <FormDescription>
                        TAG única que identificará a loja publicamente. Deve ser única no sistema.
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}
            </div>

            {/* Seção: Configurações */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-gray-900 border-b pb-2">
                Configurações
              </h3>
              <div className="grid grid-cols-1 gap-4">
                <FormField
                  control={form.control}
                  name="enable_ai_prompt_customization"
                  render={({ field }) => (
                    <FormItem className="flex flex-row items-center justify-between rounded-lg border p-4">
                      <div className="space-y-0.5">
                        <FormLabel className="text-base">
                          Customização de Prompt IA
                        </FormLabel>
                        <FormDescription>
                          Permite que o tenant customize os prompts da IA
                        </FormDescription>
                      </div>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="is_public_store"
                  render={({ field }) => (
                    <FormItem className="flex flex-row items-center justify-between rounded-lg border p-4">
                      <div className="space-y-0.5">
                        <FormLabel className="text-base">
                          Loja Pública
                        </FormLabel>
                        <FormDescription>
                          Permite acesso público ao catálogo e informações da loja sem autenticação
                        </FormDescription>
                      </div>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />
              </div>
            </div>

            {/* Seção: Administrador da Empresa */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-gray-900 border-b pb-2">
                Administrador da Empresa
              </h3>
              
              {/* Checkbox para enviar credenciais por email */}
              <FormField
                control={form.control}
                name="send_credentials_by_email"
                render={({ field }) => (
                  <FormItem className="flex flex-row items-center space-x-3 space-y-0">
                    <FormControl>
                      <Checkbox
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                    <div className="space-y-1 leading-none">
                      <FormLabel>
                        Enviar senha por email
                      </FormLabel>
                      <FormDescription>
                        Gera uma senha automática de 8 dígitos e envia por email para o administrador
                      </FormDescription>
                    </div>
                  </FormItem>
                )}
              />

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <FormField
                  control={form.control}
                  name="admin_name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Nome do Administrador</FormLabel>
                      <FormControl>
                        <Input placeholder="Ex: João Silva" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="admin_email"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Email do Administrador</FormLabel>
                      <FormControl>
                        <Input placeholder="Ex: admin@empresa.com" type="email" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="admin_phone"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Telefone *</FormLabel>
                      <FormControl>
                        <PhoneNumberInput
                          value={field.value}
                          onChange={field.onChange}
                          placeholder="Digite o número do telefone"
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                {!form.watch("send_credentials_by_email") && (
                  <FormField
                    control={form.control}
                    name="admin_password"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Senha</FormLabel>
                        <FormControl>
                          <Input placeholder="Mínimo 6 caracteres" type="password" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                )}
              </div>
            </div>

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
              >
                Cancelar
              </Button>
              <Button 
                type="submit" 
                disabled={isLoading}
                className="bg-gradient-primary"
              >
                {isLoading && (
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                )}
                Criar Empresa
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
