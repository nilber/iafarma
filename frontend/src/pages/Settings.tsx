import { useState, useEffect, useCallback } from "react";
import { Settings, User, Bell, Shield, Database, Save, Eye, EyeOff, Bot, Store, Users, MapPin, Truck, Plus, Trash2, Map, Clock, MessageSquare, Palette } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { PhoneInput } from 'react-international-phone';
import 'react-international-phone/style.css';
import { Combobox } from "@/components/ui/combobox";
import { useAuth } from "@/contexts/AuthContext";
import { useSoundNotification } from "@/contexts/SoundNotificationContext";
import { toast } from "sonner";
import { apiClient } from "@/lib/api/client";
import { useSearchParams } from "react-router-dom";
import UsersManagement from "@/components/UsersManagement";
import CoverageMap from "@/components/CoverageMap";

export default function SettingsPage() {
  const { user, updateUser } = useAuth();
  const { isEnabled: soundEnabled, toggleSound } = useSoundNotification();
  const [searchParams, setSearchParams] = useSearchParams();
  const [isLoading, setIsLoading] = useState(false);
  
  // Get initial tab from URL params
  const initialTab = searchParams.get('tab') || 'profile';
  const [activeTab, setActiveTab] = useState(initialTab);

  // Business type state - load tenant info to determine business type
  const [tenantInfo, setTenantInfo] = useState(null);
  
  // Estado para controlar o dialog de confirmação de reset
  const [showResetConfirm, setShowResetConfirm] = useState(false);
  
  // Estado para controlar o dialog de confirmação de exclusão de conta
  const [showDeleteAccountDialog, setShowDeleteAccountDialog] = useState(false);

  // Update URL when tab changes
  useEffect(() => {
    if (activeTab !== 'profile') {
      setSearchParams({ tab: activeTab });
    } else {
      setSearchParams({});
    }
  }, [activeTab, setSearchParams]);

  // Load tenant info for business type
  useEffect(() => {
    const loadTenantInfo = async () => {
      try {
        // Skip tenant API call for system_admin
        if (user?.role === 'system_admin') {
          return;
        }
        
        const tenantProfile = await apiClient.getTenantProfile();
        setTenantInfo(tenantProfile);
      } catch (error) {
        console.error("Erro ao carregar informações do tenant:", error);
      }
    };
    loadTenantInfo();
  }, [user]);


  // Load sales AI settings when prompt AI tab is activated
  useEffect(() => {
    if (activeTab === 'prompt-ai') {
      handleLoadSalesAISettings();
    }
  }, [activeTab]);



  // Load store profile when store tab is activated
  useEffect(() => {
    if (activeTab === 'store') {
      handleLoadStoreProfile();
      handleLoadDeliverySettings();
      handleLoadDeliveryZones();
      handleLoadBusinessHours();
      handleLoadEstados();
    }
  }, [activeTab]);

  const [profile, setProfile] = useState({
    name: user?.name || "",
    email: user?.email || "",
    phone: user?.phone || "",
  });

  const [notifications, setNotifications] = useState({
    emailNotifications: true,
    pushNotifications: true,
    soundNotifications: soundEnabled,
    smsNotifications: false,
    marketingEmails: false,
  });

  const [passwordData, setPasswordData] = useState({
    currentPassword: '',
    newPassword: '',
    confirmPassword: '',
  });

  const [showPasswords, setShowPasswords] = useState({
    current: false,
    new: false,
    confirm: false,
  });

  const [preferences, setPreferences] = useState({
    theme: "system",
    language: "pt-BR",
    timezone: "America/Sao_Paulo",
  });


  // Sales AI settings state
  const [salesAISettings, setSalesAISettings] = useState({
    contextLimitation: "",
    isLoading: false,
  });




  // Utility functions
  const formatCep = (value: string) => {
    // Remove tudo que não é número
    const numbers = value.replace(/\D/g, '');
    
    // Aplica a máscara XXXXX-XXX
    if (numbers.length <= 5) {
      return numbers;
    } else {
      return `${numbers.slice(0, 5)}-${numbers.slice(5, 8)}`;
    }
  };

  const cleanCep = (value: string) => {
    // Remove tudo que não é número para enviar para API
    return value.replace(/\D/g, '');
  };

  const [storeProfile, setStoreProfile] = useState({
    name: "",
    about: "",
    store_phone: "",
    isLoading: false,
  });

  // Delivery settings state
  const [deliverySettings, setDeliverySettings] = useState({
    store_street: "",
    store_number: "",
    store_neighborhood: "",
    store_city: "",
    store_state: "",
    store_zip_code: "",
    store_country: "Brasil",
    delivery_radius_km: 0,
    store_latitude: null,
    store_longitude: null,
    coordinates_configured: false,
    isLoading: false,
  });

  // Delivery zones state
  const [deliveryZones, setDeliveryZones] = useState([]);
  const [newZone, setNewZone] = useState({
    neighborhood: "",
    city: "",
    state: "",
    zone_type: "whitelist" as "whitelist" | "blacklist",
  });
  const [isLoadingZones, setIsLoadingZones] = useState(false);

  // Business hours state
  const [businessHours, setBusinessHours] = useState({
    monday: { enabled: true, open: "08:00", close: "18:00" },
    tuesday: { enabled: true, open: "08:00", close: "18:00" },
    wednesday: { enabled: true, open: "08:00", close: "18:00" },
    thursday: { enabled: true, open: "08:00", close: "18:00" },
    friday: { enabled: true, open: "08:00", close: "18:00" },
    saturday: { enabled: true, open: "08:00", close: "18:00" },
    sunday: { enabled: false, open: "08:00", close: "18:00" },
    timezone: "America/Sao_Paulo",
    isLoading: false,
  });

  // Estados para autocompletes
  const [estados, setEstados] = useState([]);
  const [cidadesLoja, setCidadesLoja] = useState([]);
  const [cidadesZona, setCidadesZona] = useState([]);
  const [loadingEstados, setLoadingEstados] = useState(false);
  const [loadingCidadesLoja, setLoadingCidadesLoja] = useState(false);
  const [loadingCidadesZona, setLoadingCidadesZona] = useState(false);

  const handleLoadStoreProfile = useCallback(async () => {
    setStoreProfile(prev => ({ ...prev, isLoading: true }));
    try {
      // Skip tenant API call for system_admin
      if (user?.role === 'system_admin') {
        return;
      }
      
      const tenantProfile = await apiClient.getTenantProfile();
      setStoreProfile(prev => ({
        ...prev,
        name: tenantProfile.name || '',
        about: tenantProfile.about || '',
        store_phone: tenantProfile.store_phone || ''
      }));
    } catch (error) {
      console.error("Erro ao carregar perfil da loja:", error);
      toast.error("Erro ao carregar perfil da loja");
    } finally {
      setStoreProfile(prev => ({ ...prev, isLoading: false }));
    }
  }, [user]);

  const handleSaveStoreProfile = async () => {
    // Validações obrigatórias
    if (!storeProfile.name.trim()) {
      toast.error("Nome da Loja é obrigatório");
      return;
    }
    
    if (!storeProfile.store_phone.trim()) {
      toast.error("Telefone da Loja é obrigatório");
      return;
    }
    
    if (!storeProfile.about.trim()) {
      toast.error("Sobre a Loja é obrigatório");
      return;
    }

    setStoreProfile(prev => ({ ...prev, isLoading: true }));
    try {
      await apiClient.updateTenantProfile({
        name: storeProfile.name,
        about: storeProfile.about,
        store_phone: storeProfile.store_phone
      });
      toast.success("Perfil da loja atualizado com sucesso!");
    } catch (error) {
      console.error("Erro ao salvar perfil da loja:", error);
      toast.error("Erro ao salvar perfil da loja");
    } finally {
      setStoreProfile(prev => ({ ...prev, isLoading: false }));
    }
  };

  // Delivery functions
  const handleLoadDeliverySettings = useCallback(async () => {
    setDeliverySettings(prev => ({ ...prev, isLoading: true }));
    try {
      const response = await apiClient.getDeliveryStoreLocation();
      setDeliverySettings(prev => ({
        ...prev,
        ...response,
      }));
    } catch (error) {
      console.error("Erro ao carregar configurações de delivery:", error);
      if (error.status !== 404) {
        toast.error("Erro ao carregar configurações de delivery");
      }
    } finally {
      setDeliverySettings(prev => ({ ...prev, isLoading: false }));
    }
  }, []);

  const handleSaveDeliverySettings = async () => {
    // Validação do raio de entrega
    if (deliverySettings.delivery_radius_km <= 0) {
      toast.error("Raio de entrega deve ser superior a 0");
      return;
    }

    if (!Number.isInteger(deliverySettings.delivery_radius_km)) {
      toast.error("Raio de entrega deve ser um número inteiro");
      return;
    }

    setDeliverySettings(prev => ({ ...prev, isLoading: true }));
    try {
      await apiClient.updateDeliveryStoreLocation({
        store_street: deliverySettings.store_street,
        store_number: deliverySettings.store_number,
        store_neighborhood: deliverySettings.store_neighborhood,
        store_city: deliverySettings.store_city,
        store_state: deliverySettings.store_state,
        store_zip_code: cleanCep(deliverySettings.store_zip_code), // Envia apenas números
        store_country: deliverySettings.store_country,
        delivery_radius_km: deliverySettings.delivery_radius_km,
      });
      toast.success("Configurações de delivery atualizadas com sucesso!");
      // Reload to get updated coordinates
      await handleLoadDeliverySettings();
    } catch (error) {
      console.error("Erro ao salvar configurações de delivery:", error);
      toast.error("Erro ao salvar configurações de delivery");
    } finally {
      setDeliverySettings(prev => ({ ...prev, isLoading: false }));
    }
  };

  const handleLoadDeliveryZones = useCallback(async () => {
    setIsLoadingZones(true);
    try {
      const response = await apiClient.getDeliveryZones();
      setDeliveryZones(response.zones || []);
    } catch (error) {
      console.error("Erro ao carregar zonas de delivery:", error);
      if (error.status !== 404) {
        toast.error("Erro ao carregar zonas de delivery");
      }
    } finally {
      setIsLoadingZones(false);
    }
  }, []);

  const handleAddDeliveryZone = async () => {
    if (!newZone.neighborhood || !newZone.city || !newZone.state) {
      toast.error("Preencha todos os campos obrigatórios");
      return;
    }

    try {
      await apiClient.manageDeliveryZone({
        ...newZone,
        action: 'add'
      });
      toast.success("Zona adicionada com sucesso!");
      setNewZone({
        neighborhood: "",
        city: "",
        state: "",
        zone_type: "whitelist" as "whitelist" | "blacklist",
      });
      await handleLoadDeliveryZones();
    } catch (error) {
      console.error("Erro ao adicionar zona:", error);
      toast.error("Erro ao adicionar zona");
    }
  };

  const handleRemoveDeliveryZone = async (zone) => {
    try {
      await apiClient.manageDeliveryZone({
        neighborhood: zone.neighborhood_name,
        city: zone.city,
        state: zone.state,
        zone_type: zone.zone_type,
        action: 'remove'
      });
      toast.success("Zona removida com sucesso!");
      await handleLoadDeliveryZones();
    } catch (error) {
      console.error("Erro ao remover zona:", error);
      toast.error("Erro ao remover zona");
    }
  };

  // Business Hours functions
  const handleLoadBusinessHours = useCallback(async () => {
    setBusinessHours(prev => ({ ...prev, isLoading: true }));
    try {
      const response = await apiClient.getTenantSettings();
      const businessHoursSetting = response.settings.find(s => s.setting_key === 'business_hours');
      
      if (businessHoursSetting?.setting_value) {
        const savedHours = JSON.parse(businessHoursSetting.setting_value);
        setBusinessHours(prev => ({ 
          ...prev, 
          ...savedHours,
          isLoading: false
        }));
      } else {
        setBusinessHours(prev => ({ ...prev, isLoading: false }));
      }
    } catch (error) {
      console.error("Erro ao carregar horários de funcionamento:", error);
      toast.error("Erro ao carregar horários de funcionamento");
      setBusinessHours(prev => ({ ...prev, isLoading: false }));
    }
  }, []);

  const handleSaveBusinessHours = async () => {
    setBusinessHours(prev => ({ ...prev, isLoading: true }));
    try {
      const { isLoading, ...hoursToSave } = businessHours;
      await apiClient.updateTenantSetting('business_hours', JSON.stringify(hoursToSave));
      toast.success("Horários de funcionamento salvos com sucesso!");
    } catch (error) {
      console.error("Erro ao salvar horários de funcionamento:", error);
      toast.error("Erro ao salvar horários de funcionamento");
    } finally {
      setBusinessHours(prev => ({ ...prev, isLoading: false }));
    }
  };

  const handleDayToggle = (day: string, enabled: boolean) => {
    setBusinessHours(prev => ({
      ...prev,
      [day]: { ...prev[day], enabled }
    }));
  };

  const handleTimeChange = (day: string, type: 'open' | 'close', time: string) => {
    setBusinessHours(prev => ({
      ...prev,
      [day]: { ...prev[day], [type]: time }
    }));
  };

    const handleCopyToAllDays = (sourceDay: string) => {
    const sourceHours = businessHours[sourceDay];
    const updatedHours = { ...businessHours };
    
    Object.keys(updatedHours).forEach(day => {
      if (day !== 'timezone' && day !== 'isLoading' && day !== sourceDay) {
        updatedHours[day] = { ...sourceHours };
      }
    });
    
    setBusinessHours(updatedHours);
    toast.success("Horários copiados para todos os dias!");
  };

  // Estados e Cidades functions
  const handleLoadEstados = useCallback(async () => {
    setLoadingEstados(true);
    try {
      const response = await apiClient.getEstados();
      setEstados(response.map(estado => ({
        value: estado.uf,
        label: `${estado.uf} - ${estado.nome}`
      })));
    } catch (error) {
      console.error("Erro ao carregar estados:", error);
      toast.error("Erro ao carregar estados");
    } finally {
      setLoadingEstados(false);
    }
  }, []);

  const handleLoadCidadesLoja = useCallback(async (uf: string) => {
    if (!uf) {
      setCidadesLoja([]);
      return;
    }
    
    setLoadingCidadesLoja(true);
    try {
      const response = await apiClient.getCidades(uf);
      setCidadesLoja(response.map(cidade => ({
        value: cidade.nome_cidade,
        label: cidade.nome_cidade
      })));
    } catch (error) {
      console.error("Erro ao carregar cidades:", error);
      toast.error("Erro ao carregar cidades");
    } finally {
      setLoadingCidadesLoja(false);
    }
  }, []);

  const handleLoadCidadesZona = useCallback(async (uf: string) => {
    if (!uf) {
      setCidadesZona([]);
      return;
    }
    
    setLoadingCidadesZona(true);
    try {
      const response = await apiClient.getCidades(uf);
      setCidadesZona(response.map(cidade => ({
        value: cidade.nome_cidade,
        label: cidade.nome_cidade
      })));
    } catch (error) {
      console.error("Erro ao carregar cidades da zona:", error);
      toast.error("Erro ao carregar cidades da zona");
    } finally {
      setLoadingCidadesZona(false);
    }
  }, []);


  // Sales AI functions
  const handleLoadSalesAISettings = useCallback(async () => {
    setSalesAISettings(prev => ({ ...prev, isLoading: true }));
    try {
      const response = await apiClient.getSalesContextLimitation();
      setSalesAISettings(prev => ({
        ...prev,
        contextLimitation: response.contextLimitation || ''
      }));
    } catch (error) {
      console.error("Erro ao carregar configurações de IA de vendas:", error);
      toast.error("Erro ao carregar configurações de IA de vendas");
    } finally {
      setSalesAISettings(prev => ({ ...prev, isLoading: false }));
    }
  }, []);

  const handleSaveSalesAISettings = async () => {
    setSalesAISettings(prev => ({ ...prev, isLoading: true }));
    try {
      await apiClient.setSalesContextLimitation(salesAISettings.contextLimitation);
      toast.success("Configurações de IA de vendas salvas!");
    } catch (error) {
      console.error("Erro ao salvar configurações de IA de vendas:", error);
      toast.error("Erro ao salvar configurações de IA de vendas");
    } finally {
      setSalesAISettings(prev => ({ ...prev, isLoading: false }));
    }
  };

  const handleResetSalesAISettings = async () => {
    setSalesAISettings(prev => ({ ...prev, isLoading: true }));
    setShowResetConfirm(false);
    try {
      await apiClient.resetSalesContextLimitation();
      // Reload the settings to get the default text
      await handleLoadSalesAISettings();
      toast.success("Configurações de IA de vendas restauradas para o padrão do sistema!");
    } catch (error) {
      console.error("Erro ao restaurar configurações de IA de vendas:", error);
      toast.error("Erro ao restaurar configurações de IA de vendas");
    } finally {
      setSalesAISettings(prev => ({ ...prev, isLoading: false }));
    }
  };


  const handleSaveProfile = async () => {
    setIsLoading(true);
    try {
      const updatedUser = await apiClient.updateProfile(profile);
      updateUser(updatedUser);
      toast.success("Perfil atualizado com sucesso!");
    } catch (error) {
      console.error('Error updating profile:', error);
      toast.error("Erro ao atualizar perfil");
    } finally {
      setIsLoading(false);
    }
  };

  const handleSaveNotifications = async () => {
    setIsLoading(true);
    try {
      // Update sound notification if changed
      if (notifications.soundNotifications !== soundEnabled) {
        toggleSound();
      }
      
      // TODO: Implement notifications update API call
      await new Promise(resolve => setTimeout(resolve, 1000)); // Simulate API call
      toast.success("Configurações de notificação atualizadas!");
    } catch (error) {
      toast.error("Erro ao atualizar configurações");
    } finally {
      setIsLoading(false);
    }
  };



  const handlePasswordChange = async () => {
    if (passwordData.newPassword !== passwordData.confirmPassword) {
      toast.error("As senhas não coincidem.");
      return;
    }

    if (passwordData.newPassword.length < 6) {
      toast.error("A nova senha deve ter pelo menos 6 caracteres.");
      return;
    }

    setIsLoading(true);
    try {
      await apiClient.changePassword({
        current_password: passwordData.currentPassword,
        new_password: passwordData.newPassword,
      });
      toast.success("Senha alterada com sucesso!");
      setPasswordData({
        currentPassword: '',
        newPassword: '',
        confirmPassword: '',
      });
    } catch (error: any) {
      console.error('Error changing password:', error);
      toast.error(error.message || "Erro ao alterar senha. Verifique sua senha atual.");
    } finally {
      setIsLoading(false);
    }
  };

  const handleDeleteAccount = async () => {
    if (!user?.tenant_id) {
      toast.error("Erro: ID do tenant não encontrado.");
      return;
    }

    setIsLoading(true);
    try {
      await apiClient.deleteTenant(user.tenant_id);
      toast.success("Conta excluída com sucesso!");
      // Redirecionar ou fazer logout após exclusão
      window.location.href = '/login';
    } catch (error: any) {
      console.error('Error deleting account:', error);
      toast.error(error.message || "Erro ao excluir conta. Tente novamente.");
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-foreground">Configurações</h1>
          <p className="text-muted-foreground">Gerencie suas preferências e configurações da conta</p>
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
        <TabsList className="flex w-full flex-wrap justify-start gap-1 h-auto p-1">
          <TabsTrigger value="profile" className="flex items-center gap-2 flex-1 min-w-fit">
            <User className="w-4 h-4" />
            Perfil
          </TabsTrigger>
          {user?.role !== 'system_admin' && (
            <TabsTrigger value="store" className="flex items-center gap-2 flex-1 min-w-fit">
              <Store className="w-4 h-4" />
              Loja
            </TabsTrigger>
          )}
          {user?.role !== 'system_admin' && (
            <TabsTrigger value="users" className="flex items-center gap-2 flex-1 min-w-fit">
              <Users className="w-4 h-4" />
              Usuários
            </TabsTrigger>
          )}
          {user?.role !== 'system_admin' && (
            <TabsTrigger value="notifications" className="flex items-center gap-2 flex-1 min-w-fit">
              <Bell className="w-4 h-4" />
              Notificações
            </TabsTrigger>
          )}
         
          {user?.role !== 'system_admin' && tenantInfo?.enable_ai_prompt_customization && (
            <TabsTrigger value="prompt-ai" className="flex items-center gap-2 flex-1 min-w-fit">
              <Bot className="w-4 h-4" />
              Prompt IA
            </TabsTrigger>
          )}


          <TabsTrigger value="security" className="flex items-center gap-2 flex-1 min-w-fit">
            <Shield className="w-4 h-4" />
            Segurança
          </TabsTrigger>
        </TabsList>

        <TabsContent value="profile">
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <div className="lg:col-span-2">
              <Card className="border-0 shadow-custom-md">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <User className="w-5 h-5 text-primary" />
                    Informações do Perfil
                  </CardTitle>
                  <CardDescription>
                    Atualize suas informações pessoais
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="name">Nome completo</Label>
                      <Input
                        id="name"
                        value={profile.name}
                        onChange={(e) => setProfile({ ...profile, name: e.target.value })}
                        placeholder="Seu nome completo"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="email">Email</Label>
                      <Input
                        id="email"
                        type="email"
                        value={profile.email}
                        readOnly
                        className="bg-muted text-muted-foreground cursor-not-allowed"
                        placeholder="seu@email.com"
                      />
                      <p className="text-xs text-muted-foreground">
                        O email não pode ser alterado. Entre em contato com o suporte se necessário.
                      </p>
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="phone">Telefone</Label>
                    <PhoneInput
                      defaultCountry="br"
                      value={profile.phone}
                      onChange={(phone) => setProfile({ ...profile, phone })}
                      inputClassName="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                      countrySelectorStyleProps={{
                        buttonClassName: "border border-input bg-background hover:bg-accent hover:text-accent-foreground h-10 px-3 rounded-l-md"
                      }}
                    />
                  </div>
                  <Button onClick={handleSaveProfile} disabled={isLoading} className="bg-gradient-primary">
                    <Save className="w-4 h-4 mr-2" />
                    {isLoading ? "Salvando..." : "Salvar Perfil"}
                  </Button>
                </CardContent>
              </Card>
            </div>

            <div className="space-y-6">
              <Card className="border-0 shadow-custom-md">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Shield className="w-5 h-5 text-primary" />
                    Informações da Conta
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div>
                    <Label className="text-sm font-medium">Role</Label>
                    <p className="text-sm text-muted-foreground capitalize">
                      {user?.role?.replace('_', ' ')}
                    </p>
                  </div>
                  {user?.tenant_id && (
                    <div>
                      <Label className="text-sm font-medium">Tenant ID</Label>
                      <p className="text-xs text-muted-foreground font-mono">
                        {user.tenant_id}
                      </p>
                    </div>
                  )}
                  <div>
                    <Label className="text-sm font-medium">Último Login</Label>
                    <p className="text-sm text-muted-foreground">
                      {user?.last_login_at ? new Date(user.last_login_at).toLocaleString('pt-BR') : 'Nunca'}
                    </p>
                  </div>
                </CardContent>
              </Card>
            </div>
          </div>
        </TabsContent>

        <TabsContent value="notifications">
          <Card className="border-0 shadow-custom-md">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Bell className="w-5 h-5 text-primary" />
                Preferências de Notificação
              </CardTitle>
              <CardDescription>
                Configure como você deseja ser notificado
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>Alertas Sonoros</Label>
                  <p className="text-sm text-muted-foreground">
                    Reproduzir som quando receber novas mensagens
                  </p>
                </div>
                <Switch
                  checked={notifications.soundNotifications}
                  onCheckedChange={(checked) => 
                    setNotifications({ ...notifications, soundNotifications: checked })
                  }
                />
              </div>
              
              <Separator />

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>Notificações por Email</Label>
                  <p className="text-sm text-muted-foreground">
                    Receber notificações importantes por email
                  </p>
                </div>
                <Switch
                  checked={notifications.emailNotifications}
                  onCheckedChange={(checked) => 
                    setNotifications({ ...notifications, emailNotifications: checked })
                  }
                />
              </div>
              
              <Separator />
              
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>Notificações Push</Label>
                  <p className="text-sm text-muted-foreground">
                    Receber notificações no navegador
                  </p>
                </div>
                <Switch
                  checked={notifications.pushNotifications}
                  onCheckedChange={(checked) => 
                    setNotifications({ ...notifications, pushNotifications: checked })
                  }
                />
              </div>
              
              <Separator />
              
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>SMS</Label>
                  <p className="text-sm text-muted-foreground">
                    Receber notificações por SMS
                  </p>
                </div>
                <Switch
                  checked={notifications.smsNotifications}
                  onCheckedChange={(checked) => 
                    setNotifications({ ...notifications, smsNotifications: checked })
                  }
                />
              </div>
              
              <Separator />
              
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>Emails de Marketing</Label>
                  <p className="text-sm text-muted-foreground">
                    Receber dicas, novidades e promoções
                  </p>
                </div>
                <Switch
                  checked={notifications.marketingEmails}
                  onCheckedChange={(checked) => 
                    setNotifications({ ...notifications, marketingEmails: checked })
                  }
                />
              </div>
              
              <Separator />
              
              <Button onClick={handleSaveNotifications} disabled={isLoading} className="bg-gradient-primary">
                <Save className="w-4 h-4 mr-2" />
                {isLoading ? "Salvando..." : "Salvar Notificações"}
              </Button>
            </CardContent>
          </Card>
        </TabsContent>



        <TabsContent value="store">
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <div className="lg:col-span-2">
              <Card className="border-0 shadow-custom-md">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Store className="w-5 h-5 text-primary" />
                    Perfil da Loja
                  </CardTitle>
                  <CardDescription>
                    Configure as informações da sua loja para personalizar a experiência da IA
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="storeName">Nome da Loja</Label>
                    <Input
                      id="storeName"
                      value={storeProfile.name}
                      onChange={(e) => setStoreProfile({ ...storeProfile, name: e.target.value })}
                      placeholder="Nome da sua loja"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="storePhone">Telefone da Loja</Label>
                    <PhoneInput
                      defaultCountry="br"
                      value={storeProfile.store_phone}
                      onChange={(phone) => setStoreProfile({ ...storeProfile, store_phone: phone })}
                      inputClassName="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                      countrySelectorStyleProps={{
                        buttonClassName: "border border-input bg-background hover:bg-accent hover:text-accent-foreground h-10 px-3 rounded-l-md"
                      }}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="storeAbout">Sobre a Loja</Label>
                    <Textarea
                      id="storeAbout"
                      value={storeProfile.about}
                      onChange={(e) => setStoreProfile({ ...storeProfile, about: e.target.value })}
                      placeholder="Descreva sua loja, produtos, serviços e diferenciais. Esta informação ajudará a IA a fornecer respostas mais personalizadas."
                      rows={4}
                    />
                  </div>
                  <div className="bg-muted/50 p-4 rounded-lg">
                    <h4 className="font-medium mb-2">Dicas para o campo "Sobre":</h4>
                    <ul className="text-sm text-muted-foreground space-y-1">
                      <li>• Mencione o tipo de produtos/serviços que oferece</li>
                      <li>• Inclua diferenciais e especializações</li>
                      <li>• Descreva o público-alvo</li>
                      <li>• Adicione informações relevantes para atendimento</li>
                    </ul>
                  </div>
                  <Button 
                    onClick={handleSaveStoreProfile} 
                    disabled={storeProfile.isLoading}
                    className="bg-gradient-primary"
                  >
                    <Save className="w-4 h-4 mr-2" />
                    {storeProfile.isLoading ? "Salvando..." : "Salvar Perfil da Loja"}
                  </Button>
                </CardContent>
              </Card>
            </div>

            <div className="space-y-6">
              <Card className="border-0 shadow-custom-md">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Bot className="w-5 h-5 text-primary" />
                    Como a IA Usa Essas Informações
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="text-sm text-muted-foreground space-y-2">
                    <p>
                      <strong>Nome da Loja:</strong> Usado para personalizar as respostas da IA, 
                      mencionando o nome da sua empresa nas interações.
                    </p>
                    <p>
                      <strong>Sobre a Loja:</strong> Fornece contexto para a IA entender melhor 
                      seu negócio e dar respostas mais precisas e relevantes.
                    </p>
                  </div>
                  <Separator />
                  <div className="text-sm">
                    <p className="font-medium mb-2">Carregamento Automático:</p>
                    <p className="text-muted-foreground">
                      As informações são carregadas automaticamente quando você acessa esta aba.
                    </p>
                  </div>
                </CardContent>
              </Card>
            </div>
          </div>

          {/* Delivery Settings Section */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-6">
            {/* Store Location Settings */}
            <Card className="border-0 shadow-custom-md">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <MapPin className="w-5 h-5 text-primary" />
                  Endereço da Loja
                </CardTitle>
                <CardDescription>
                  Configure o endereço da sua loja para cálculo de entregas e geolocalização
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="store_street">Rua *</Label>
                    <Input
                      id="store_street"
                      value={deliverySettings.store_street}
                      onChange={(e) => setDeliverySettings({ ...deliverySettings, store_street: e.target.value })}
                      placeholder="Nome da rua"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="store_number">Número</Label>
                    <Input
                      id="store_number"
                      value={deliverySettings.store_number}
                      onChange={(e) => setDeliverySettings({ ...deliverySettings, store_number: e.target.value })}
                      placeholder="123"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="store_neighborhood">Bairro</Label>
                    <Input
                      id="store_neighborhood"
                      value={deliverySettings.store_neighborhood}
                      onChange={(e) => setDeliverySettings({ ...deliverySettings, store_neighborhood: e.target.value })}
                      placeholder="Nome do bairro"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="store_state">Estado *</Label>
                    <Combobox
                      options={estados}
                      value={deliverySettings.store_state}
                      placeholder="Selecione o estado..."
                      searchPlaceholder="Buscar estado..."
                      emptyText="Nenhum estado encontrado."
                      onValueChange={(value) => {
                        setDeliverySettings({ ...deliverySettings, store_state: value, store_city: "" });
                        handleLoadCidadesLoja(value);
                      }}
                      disabled={loadingEstados}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="store_city">Cidade *</Label>
                    <Combobox
                      options={cidadesLoja}
                      value={deliverySettings.store_city}
                      placeholder="Selecione a cidade..."
                      searchPlaceholder="Buscar cidade..."
                      emptyText="Nenhuma cidade encontrada."
                      onValueChange={(value) => setDeliverySettings({ ...deliverySettings, store_city: value })}
                      disabled={loadingCidadesLoja || !deliverySettings.store_state}
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="store_zip_code">CEP</Label>
                    <Input
                      id="store_zip_code"
                      value={formatCep(deliverySettings.store_zip_code)}
                      onChange={(e) => setDeliverySettings({ ...deliverySettings, store_zip_code: formatCep(e.target.value) })}
                      placeholder="00000-000"
                      maxLength={9}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="delivery_radius_km">Raio de Entrega (km)</Label>
                    <Input
                      id="delivery_radius_km"
                      type="number"
                      min="0"
                      step="0.5"
                      value={deliverySettings.delivery_radius_km}
                      onChange={(e) => setDeliverySettings({ ...deliverySettings, delivery_radius_km: parseFloat(e.target.value) || 0 })}
                      placeholder="0 = sem limite"
                    />
                  </div>
                </div>

                {deliverySettings.coordinates_configured && (
                  <div className="space-y-4">
                    <div className="bg-green-50 border border-green-200 rounded-lg p-3">
                      <div className="flex items-center gap-2 text-green-700">
                        <Map className="w-4 h-4" />
                        <span className="font-medium">Coordenadas Configuradas</span>
                      </div>
                      <p className="text-sm text-green-600 mt-1">
                        Lat: {deliverySettings.store_latitude}, Lng: {deliverySettings.store_longitude}
                      </p>
                    </div>

                    {/* Coverage Map */}
                    <div className="space-y-2">
                      <Label className="text-sm font-medium">Área de Cobertura</Label>
                      <CoverageMap
                        latitude={deliverySettings.store_latitude}
                        longitude={deliverySettings.store_longitude}
                        radiusKm={deliverySettings.delivery_radius_km}
                        storeName={storeProfile.name || "Loja"}
                        className="border-muted"
                      />
                      {deliverySettings.delivery_radius_km > 0 ? (
                        <p className="text-xs text-muted-foreground">
                          Área de cobertura: {deliverySettings.delivery_radius_km}km de raio a partir da sua loja
                        </p>
                      ) : (
                        <p className="text-xs text-muted-foreground">
                          Sem limite de raio de entrega configurado
                        </p>
                      )}
                    </div>
                  </div>
                )}

                <Button 
                  onClick={handleSaveDeliverySettings} 
                  disabled={deliverySettings.isLoading}
                  className="bg-gradient-primary w-full"
                >
                  <Save className="w-4 h-4 mr-2" />
                  {deliverySettings.isLoading ? "Salvando..." : "Salvar Endereço"}
                </Button>
              </CardContent>
            </Card>

            {/* Delivery Zones Management */}
            <Card className="border-0 shadow-custom-md">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Truck className="w-5 h-5 text-primary" />
                  Zonas de Entrega
                </CardTitle>
                <CardDescription>
                  Gerencie áreas específicas para entrega (whitelist/blacklist)
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                {/* Add new zone */}
                <div className="space-y-3 p-4 border rounded-lg bg-muted/30">
                  <h4 className="font-medium">Adicionar Nova Zona</h4>
                  <div className="grid grid-cols-1 gap-3">
                    <div className="space-y-2">
                      <Label htmlFor="new_neighborhood">Bairro *</Label>
                      <Input
                        id="new_neighborhood"
                        value={newZone.neighborhood}
                        onChange={(e) => setNewZone({ ...newZone, neighborhood: e.target.value })}
                        placeholder="Nome do bairro"
                      />
                    </div>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                      <div className="space-y-2">
                        <Label htmlFor="new_state">Estado *</Label>
                        <Combobox
                          options={estados}
                          value={newZone.state}
                          placeholder="Selecione o estado..."
                          searchPlaceholder="Buscar estado..."
                          emptyText="Nenhum estado encontrado."
                          onValueChange={(value) => {
                            setNewZone({ ...newZone, state: value, city: "" });
                            handleLoadCidadesZona(value);
                          }}
                          disabled={loadingEstados}
                        />
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="new_city">Cidade *</Label>
                        <Combobox
                          options={cidadesZona}
                          value={newZone.city}
                          placeholder="Selecione a cidade..."
                          searchPlaceholder="Buscar cidade..."
                          emptyText="Nenhuma cidade encontrada."
                          onValueChange={(value) => setNewZone({ ...newZone, city: value })}
                          disabled={loadingCidadesZona || !newZone.state}
                        />
                      </div>
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="zone_type">Tipo de Zona</Label>
                      <Select value={newZone.zone_type} onValueChange={(value: "whitelist" | "blacklist") => setNewZone({ ...newZone, zone_type: value })}>
                        <SelectTrigger>
                          <SelectValue placeholder="Selecione o tipo" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="whitelist">Whitelist (Permitir sempre)</SelectItem>
                          <SelectItem value="blacklist">Blacklist (Bloquear sempre)</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <Button onClick={handleAddDeliveryZone} className="w-full">
                      <Plus className="w-4 h-4 mr-2" />
                      Adicionar Zona
                    </Button>
                  </div>
                </div>

                {/* Existing zones list */}
                <div className="space-y-2">
                  <h4 className="font-medium">Zonas Configuradas</h4>
                  {isLoadingZones ? (
                    <p className="text-sm text-muted-foreground">Carregando zonas...</p>
                  ) : deliveryZones.length === 0 ? (
                    <p className="text-sm text-muted-foreground">Nenhuma zona configurada</p>
                  ) : (
                    <div className="space-y-2 max-h-64 overflow-y-auto">
                      {deliveryZones.map((zone: any, index: number) => (
                        <div key={index} className="flex items-center justify-between p-3 border rounded-lg">
                          <div className="flex-1">
                            <div className="font-medium">{zone.neighborhood_name}</div>
                            <div className="text-sm text-muted-foreground">
                              {zone.city}, {zone.state}
                            </div>
                            <div className={`text-xs px-2 py-1 rounded-full inline-block mt-1 ${
                              zone.zone_type === 'whitelist' 
                                ? 'bg-green-100 text-green-700' 
                                : 'bg-red-100 text-red-700'
                            }`}>
                              {zone.zone_type === 'whitelist' ? 'Whitelist' : 'Blacklist'}
                            </div>
                          </div>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleRemoveDeliveryZone(zone)}
                            className="text-red-600 hover:text-red-700"
                          >
                            <Trash2 className="w-4 h-4" />
                          </Button>
                        </div>
                      ))}
                    </div>
                  )}
                </div>

                <div className="bg-blue-50 border border-blue-200 rounded-lg p-3">
                  <div className="text-sm text-blue-800">
                    <p className="font-medium mb-1">Como funcionam as zonas:</p>
                    <ul className="space-y-1 text-blue-700">
                      <li>• <strong>Whitelist:</strong> Áreas onde sempre é permitido entregar</li>
                      <li>• <strong>Blacklist:</strong> Áreas onde nunca é permitido entregar</li>
                      <li>• Se não estiver em nenhuma lista, verifica o raio de entrega</li>
                    </ul>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Business Hours Section */}
          <div className="grid grid-cols-1 gap-6 mt-6">
            <Card className="border-0 shadow-custom-md">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Clock className="w-5 h-5 text-primary" />
                  Horários de Funcionamento
                </CardTitle>
                <CardDescription>
                  Configure os dias e horários de funcionamento da sua loja. A IA usará essas informações para informar os clientes quando a loja está fechada.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                {businessHours.isLoading ? (
                  <p className="text-sm text-muted-foreground">Carregando horários...</p>
                ) : (
                  <>
                    {/* Timezone Selection */}
                    <div className="space-y-2">
                      <Label htmlFor="timezone">Fuso Horário</Label>
                      <Select 
                        value={businessHours.timezone} 
                        onValueChange={(value) => setBusinessHours(prev => ({ ...prev, timezone: value }))}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Selecione o fuso horário" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="America/Sao_Paulo">Brasil - São Paulo (GMT-3)</SelectItem>
                          <SelectItem value="America/Manaus">Brasil - Manaus (GMT-4)</SelectItem>
                          <SelectItem value="America/Rio_Branco">Brasil - Rio Branco (GMT-5)</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>

                    <Separator />

                    {/* Days Configuration */}
                    <div className="space-y-4">
                      <div className="flex items-center justify-between">
                        <h4 className="font-medium">Configuração por Dia</h4>
                        <div className="text-sm text-muted-foreground">
                          {new Date().toLocaleString('pt-BR', { 
                            timeZone: businessHours.timezone,
                            weekday: 'long',
                            hour: '2-digit',
                            minute: '2-digit'
                          })}
                        </div>
                      </div>

                      {[
                        { key: 'monday', label: 'Segunda-feira' },
                        { key: 'tuesday', label: 'Terça-feira' },
                        { key: 'wednesday', label: 'Quarta-feira' },
                        { key: 'thursday', label: 'Quinta-feira' },
                        { key: 'friday', label: 'Sexta-feira' },
                        { key: 'saturday', label: 'Sábado' },
                        { key: 'sunday', label: 'Domingo' }
                      ].map(({ key, label }) => (
                        <div key={key} className="grid grid-cols-12 gap-4 items-center p-4 border rounded-lg bg-muted/30">
                          <div className="col-span-3">
                            <div className="flex items-center space-x-2">
                              <Switch
                                checked={businessHours[key]?.enabled || false}
                                onCheckedChange={(enabled) => handleDayToggle(key, enabled)}
                              />
                              <Label className="font-medium">{label}</Label>
                            </div>
                          </div>
                          
                          {businessHours[key]?.enabled && (
                            <>
                              <div className="col-span-3">
                                <div className="space-y-1">
                                  <Label className="text-xs text-muted-foreground">Abertura</Label>
                                  <Input
                                    type="time"
                                    value={businessHours[key]?.open || '08:00'}
                                    onChange={(e) => handleTimeChange(key, 'open', e.target.value)}
                                    className="text-sm"
                                  />
                                </div>
                              </div>
                              <div className="col-span-3">
                                <div className="space-y-1">
                                  <Label className="text-xs text-muted-foreground">Fechamento</Label>
                                  <Input
                                    type="time"
                                    value={businessHours[key]?.close || '18:00'}
                                    onChange={(e) => handleTimeChange(key, 'close', e.target.value)}
                                    className="text-sm"
                                  />
                                </div>
                              </div>
                              <div className="col-span-3">
                                <Button
                                  variant="outline"
                                  size="sm"
                                  onClick={() => handleCopyToAllDays(key)}
                                  className="w-full text-xs"
                                >
                                  Copiar para todos
                                </Button>
                              </div>
                            </>
                          )}
                          
                          {!businessHours[key]?.enabled && (
                            <div className="col-span-9 flex items-center">
                              <span className="text-sm text-muted-foreground">Fechado</span>
                            </div>
                          )}
                        </div>
                      ))}
                    </div>

                    <Separator />

                    {/* Info Box */}
                    <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                      <div className="text-sm text-blue-800">
                        <p className="font-medium mb-2">Como a IA usa esses horários:</p>
                        <ul className="space-y-1 text-blue-700">
                          <li>• Verifica automaticamente se a loja está aberta no momento da conversa</li>
                          <li>• Informa aos clientes quando a loja está fechada</li>
                          <li>• Pode sugerir horários de retorno quando fechada</li>
                          <li>• Considera o fuso horário configurado para cálculos precisos</li>
                        </ul>
                      </div>
                    </div>

                    {/* Save Button */}
                    <Button 
                      onClick={handleSaveBusinessHours}
                      disabled={businessHours.isLoading}
                      className="bg-gradient-primary w-full"
                    >
                      <Save className="w-4 h-4 mr-2" />
                      {businessHours.isLoading ? "Salvando..." : "Salvar Horários de Funcionamento"}
                    </Button>
                  </>
                )}
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="users">
          <UsersManagement />
        </TabsContent>


        <TabsContent value="prompt-ai">
          <div className="grid grid-cols-1 gap-6">
            <Card className="border-0 shadow-custom-md">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Bot className="w-5 h-5 text-primary" />
                  Configurações de Prompt IA
                </CardTitle>
                <CardDescription>
                  Configure as limitações de contexto e comportamentos personalizados da IA
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="salesContextLimitation">Limitação de Contexto</Label>
                    <p className="text-sm text-muted-foreground">
                      Configure instruções específicas sobre limitações e comportamentos que a IA deve seguir. 
                      Estas instruções complementam o prompt principal e não substituem as funcionalidades básicas do sistema.
                    </p>
                    <Textarea
                      id="salesContextLimitation"
                      className="min-h-[200px]"
                      value={salesAISettings.contextLimitation}
                      onChange={(e) => setSalesAISettings({ ...salesAISettings, contextLimitation: e.target.value })}
                      placeholder="Ex: Não ofereça produtos em promoção sem confirmar disponibilidade. Sempre pergunte se o cliente tem alguma preferência específica antes de sugerir produtos. Limite ofertas a até 3 produtos por consulta..."
                    />
                  </div>
                  
                  <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                    <div className="text-sm text-blue-800">
                      <p className="font-medium mb-1">ℹ️ Sobre as Limitações de Contexto:</p>
                      <ul className="space-y-1 text-xs">
                        <li>• Estas instruções são adicionadas ao prompt principal da IA</li>
                        <li>• Use para definir comportamentos específicos do seu negócio</li>
                        <li>• Não interferem nas funcionalidades básicas (buscar produtos, adicionar ao carrinho, etc.)</li>
                        <li>• Ideal para definir políticas de vendas, restrições ou orientações especiais</li>
                      </ul>
                    </div>
                  </div>
                  
                  <Separator />
                  
                  <div className="flex gap-3">
                    <Button 
                      onClick={handleSaveSalesAISettings} 
                      disabled={salesAISettings.isLoading} 
                      className="bg-gradient-primary"
                    >
                      <Save className="w-4 h-4 mr-2" />
                      {salesAISettings.isLoading ? "Salvando..." : "Salvar Configurações"}
                    </Button>
                    
                    <AlertDialog open={showResetConfirm} onOpenChange={setShowResetConfirm}>
                      <AlertDialogTrigger asChild>
                        <Button 
                          variant="outline" 
                          disabled={salesAISettings.isLoading}
                          className="text-orange-600 hover:text-orange-700"
                        >
                          {salesAISettings.isLoading ? "Restaurando..." : "Restaurar Padrão"}
                        </Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>Confirmar Restauração</AlertDialogTitle>
                          <AlertDialogDescription className="space-y-2">
                            <p>
                              <strong>⚠️ Atenção:</strong> Esta ação irá <strong>apagar todos os dados atuais</strong> das configurações de prompt da IA.
                            </p>
                            <p>
                              As configurações serão restauradas para o <strong>padrão do sistema</strong> e todas as personalizações atuais serão perdidas permanentemente.
                            </p>
                            <p className="text-sm text-muted-foreground">
                              Esta ação não pode ser desfeita. Certifique-se de que deseja continuar.
                            </p>
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>Cancelar</AlertDialogCancel>
                          <AlertDialogAction 
                            onClick={handleResetSalesAISettings}
                            className="bg-orange-600 hover:bg-orange-700"
                          >
                            Confirmar Restauração
                          </AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>



        <TabsContent value="security">
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <div className="lg:col-span-2">
              <Card className="border-0 shadow-custom-md">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Shield className="w-5 h-5 text-primary" />
                    Alterar Senha
                  </CardTitle>
                  <CardDescription>
                    Altere sua senha para manter sua conta segura
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="currentPassword">Senha atual</Label>
                    <div className="relative">
                      <Input
                        id="currentPassword"
                        type={showPasswords.current ? "text" : "password"}
                        value={passwordData.currentPassword}
                        onChange={(e) => setPasswordData({ ...passwordData, currentPassword: e.target.value })}
                        placeholder="Digite sua senha atual"
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                        onClick={() => setShowPasswords({ ...showPasswords, current: !showPasswords.current })}
                      >
                        {showPasswords.current ? (
                          <EyeOff className="h-4 w-4" />
                        ) : (
                          <Eye className="h-4 w-4" />
                        )}
                      </Button>
                    </div>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="newPassword">Nova senha</Label>
                    <div className="relative">
                      <Input
                        id="newPassword"
                        type={showPasswords.new ? "text" : "password"}
                        value={passwordData.newPassword}
                        onChange={(e) => setPasswordData({ ...passwordData, newPassword: e.target.value })}
                        placeholder="Digite sua nova senha"
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                        onClick={() => setShowPasswords({ ...showPasswords, new: !showPasswords.new })}
                      >
                        {showPasswords.new ? (
                          <EyeOff className="h-4 w-4" />
                        ) : (
                          <Eye className="h-4 w-4" />
                        )}
                      </Button>
                    </div>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="confirmPassword">Confirmar nova senha</Label>
                    <div className="relative">
                      <Input
                        id="confirmPassword"
                        type={showPasswords.confirm ? "text" : "password"}
                        value={passwordData.confirmPassword}
                        onChange={(e) => setPasswordData({ ...passwordData, confirmPassword: e.target.value })}
                        placeholder="Confirme sua nova senha"
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                        onClick={() => setShowPasswords({ ...showPasswords, confirm: !showPasswords.confirm })}
                      >
                        {showPasswords.confirm ? (
                          <EyeOff className="h-4 w-4" />
                        ) : (
                          <Eye className="h-4 w-4" />
                        )}
                      </Button>
                    </div>
                  </div>

                  <div className="bg-muted/50 p-4 rounded-lg">
                    <h4 className="font-medium mb-2">Requisitos da senha:</h4>
                    <ul className="text-sm text-muted-foreground space-y-1">
                      <li>• Mínimo de 6 caracteres</li>
                      <li>• Recomendado: Misture letras, números e símbolos</li>
                      <li>• Evite usar informações pessoais</li>
                    </ul>
                  </div>

                  <Button 
                    onClick={handlePasswordChange} 
                    disabled={isLoading || !passwordData.currentPassword || !passwordData.newPassword || !passwordData.confirmPassword}
                    className="bg-gradient-primary"
                  >
                    <Save className="w-4 h-4 mr-2" />
                    {isLoading ? "Alterando..." : "Alterar Senha"}
                  </Button>
                </CardContent>
              </Card>
            </div>

            <div className="space-y-6">
              <Card className="border-0 shadow-custom-md">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Database className="w-5 h-5 text-primary" />
                    Dados e Privacidade
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <Button variant="outline" className="w-full">
                    Exportar Dados
                  </Button>
                  <Button variant="outline" className="w-full">
                    Política de Privacidade
                  </Button>
                  <AlertDialog open={showDeleteAccountDialog} onOpenChange={setShowDeleteAccountDialog}>
                    <AlertDialogTrigger asChild>
                      <Button variant="destructive" className="w-full">
                        Excluir Conta
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>Confirmar exclusão da conta</AlertDialogTitle>
                        <AlertDialogDescription>
                          Esta ação não pode ser desfeita. Todos os dados da sua conta serão permanentemente removidos, incluindo:
                          <br /><br />
                          • Todas as conversas e mensagens
                          <br />
                          • Configurações e personalizações
                          <br />
                          • Histórico de pedidos
                          <br />
                          • Dados de clientes e produtos
                          <br /><br />
                          Tem certeza que deseja excluir sua conta?
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>Cancelar</AlertDialogCancel>
                        <AlertDialogAction
                          onClick={handleDeleteAccount}
                          className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                          disabled={isLoading}
                        >
                          {isLoading ? "Excluindo..." : "Sim, excluir conta"}
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </CardContent>
              </Card>
            </div>
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}
