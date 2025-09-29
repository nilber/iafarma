import { useEffect, useState } from "react";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { useUpdateTenant, useTenantUsersForAdmin, useUpdateTenantAdmin, usePlans } from "@/lib/api/hooks";
import { toast } from "@/hooks/use-toast";
import { Loader2 } from "lucide-react";
import { Tenant, User, Plan } from "@/lib/api/types";

const formSchema = z.object({
  // Tenant fields
  name: z.string().min(2, "Nome deve ter pelo menos 2 caracteres"),
  domain: z.string().optional().or(z.literal("")),
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
  admin_name: z.string().min(2, "Nome do admin deve ter pelo menos 2 caracteres"),
  admin_email: z.string().email("Email inválido"),
  admin_phone: z.string().optional(),
  admin_password: z.string().optional(),
}).refine((data) => {
  // Se is_public_store for true, tag deve ser obrigatório e não vazio
  if (data.is_public_store && (!data.tag || data.tag.trim() === "")) {
    return false;
  }
  return true;
}, {
  message: "TAG é obrigatório para lojas públicas",
  path: ["tag"],
});

type FormData = z.infer<typeof formSchema>;

interface EditTenantDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  tenant: Tenant | null;
}

export function EditTenantDialog({ open, onOpenChange, tenant }: EditTenantDialogProps) {
  const [isLoading, setIsLoading] = useState(false);
  const [tenantAdmin, setTenantAdmin] = useState<User | null>(null);
  const { data: plans = [], isLoading: loadingPlans } = usePlans();
  
  const form = useForm<FormData>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
      domain: "",
      plan_id: "",
      status: "active",
      business_type: "sales",
      business_category: "loja",
      cost_per_message: 0,
      enable_ai_prompt_customization: false,
      is_public_store: false,
      admin_name: "",
      admin_email: "",
      admin_phone: "",
      admin_password: "",
    },
  });

  const updateTenantMutation = useUpdateTenant();
  const updateTenantAdminMutation = useUpdateTenantAdmin();
  
  // Load tenant users when dialog opens
  const { data: tenantUsersData, isLoading: loadingUsers } = useTenantUsersForAdmin(
    tenant?.id || "",
    { limit: 50 }
  );

  // Reset form when tenant changes
  useEffect(() => {
    if (tenant && tenantUsersData) {
      // Find tenant admin user
      const adminUser = tenantUsersData.users.find(user => user.role === 'tenant_admin');
      setTenantAdmin(adminUser || null);
      
      // Map tenant status to valid enum values
      const validStatuses = ['active', 'inactive', 'suspended'];
      const tenantStatus = validStatuses.includes(tenant.status) ? tenant.status : 'active';
      
      // Map tenant business_type to valid enum values
      const validBusinessTypes = ['sales'];
      const businessType = validBusinessTypes.includes(tenant.business_type) ? tenant.business_type : 'sales';
      
      form.reset({
        name: tenant.name,
        domain: tenant.domain || "",
        plan_id: tenant.plan_id || "",
        status: tenantStatus as any,
        business_type: businessType as any,
        business_category: tenant.business_category || "loja",
        cost_per_message: tenant.cost_per_message ?? 0,
        enable_ai_prompt_customization: tenant.enable_ai_prompt_customization ?? false,
        is_public_store: (tenant as any).is_public_store ?? false,
        tag: tenant.tag || "",
        admin_name: adminUser?.name || "",
        admin_email: adminUser?.email || "",
        admin_phone: adminUser?.phone || "",
        admin_password: "",
      });
    }
  }, [tenant, tenantUsersData, form]);

  const onSubmit = async (data: FormData) => {
    if (!tenant || isLoading) return; // Adiciona proteção extra contra múltiplas execuções
    
    setIsLoading(true);
    try {
      // Update tenant data
      const tenantPayload = {
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
        // max_users and max_messages now come from plan relationship
      };
      
      console.log('🏢 Updating tenant with payload:', tenantPayload);
      console.log('🏢 Tenant ID:', tenant.id);
      await updateTenantMutation.mutateAsync({ 
        id: tenant.id, 
        tenant: tenantPayload
      });

      // Update tenant admin if exists AND admin data has changed
      if (tenantAdmin) {
        const adminNeedsUpdate = 
          data.admin_name !== (tenantAdmin.name || "") ||
          data.admin_email !== (tenantAdmin.email || "") ||
          data.admin_phone !== (tenantAdmin.phone || "") ||
          (data.admin_password && data.admin_password.trim() !== "");

        if (adminNeedsUpdate) {
          const adminData: any = {
            name: data.admin_name,
            email: data.admin_email,
            phone: data.admin_phone || "",
          };
          
          // Only include password if provided
          if (data.admin_password && data.admin_password.trim() !== "") {
            adminData.password = data.admin_password;
          }

          console.log('👤 Updating tenant admin with payload:', adminData);
          console.log('👤 Admin ID:', tenantAdmin.id);
          await updateTenantAdminMutation.mutateAsync({
            userId: tenantAdmin.id,
            data: adminData,
          });
        } else {
          console.log('👤 No admin changes detected, skipping admin update');
        }
      }
      
      toast({
        title: "Sucesso",
        description: "Empresa atualizada com sucesso!",
      });
      
      onOpenChange(false);
    } catch (error: any) {
      console.error('❌ Error updating tenant/admin:', error);
      toast({
        title: "Erro",
        description: error.message || "Erro ao atualizar empresa",
        variant: "destructive",
      });
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Editar Empresa</DialogTitle>
          <DialogDescription>
            Edite as informações da empresa
          </DialogDescription>
        </DialogHeader>
        
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
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
                  <Select onValueChange={field.onChange} value={field.value || ""}>
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
                  <Select onValueChange={field.onChange} value={field.value}>
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
                  <Select onValueChange={field.onChange} value={field.value}>
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
                  <Select onValueChange={field.onChange} value={field.value}>
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

            <FormField
              control={form.control}
              name="cost_per_message"
              render={({ field }) => (
                <FormItem>
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

            <FormField
              control={form.control}
              name="enable_ai_prompt_customization"
              render={({ field }) => (
                <FormItem className="flex flex-row items-center justify-between rounded-lg border p-4">
                  <div className="space-y-0.5">
                    <FormLabel className="text-base">
                      Personalização de Prompt IA
                    </FormLabel>
                    <FormDescription>
                      Permite que o tenant customize os prompts da IA para suas necessidades específicas
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

            {form.watch("is_public_store") && (
              <FormField
                control={form.control}
                name="tag"
                render={({ field }) => (
                  <FormItem>
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

            {/* Separador */}
            <div className="border-t pt-4">
              <h3 className="text-lg font-semibold mb-4">Administrador da Empresa</h3>
              
              {loadingUsers ? (
                <div className="flex items-center justify-center py-4">
                  <Loader2 className="w-6 h-6 animate-spin" />
                  <span className="ml-2">Carregando dados do administrador...</span>
                </div>
              ) : !tenantAdmin ? (
                <div className="bg-yellow-50 border border-yellow-200 rounded-md p-4 mb-4">
                  <p className="text-yellow-800">
                    ⚠️ Nenhum administrador encontrado para esta empresa. 
                    Considere criar um através do diálogo "Nova Empresa".
                  </p>
                </div>
              ) : (
                <>
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
                        <FormLabel>Telefone (opcional)</FormLabel>
                        <FormControl>
                          <Input placeholder="Ex: +5511999887766" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name="admin_password"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Nova Senha (opcional)</FormLabel>
                        <FormControl>
                          <Input 
                            placeholder="Deixe em branco para manter a senha atual" 
                            type="password" 
                            {...field} 
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </>
              )}
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
                Salvar Alterações
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
