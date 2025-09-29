import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useSoundNotification } from "@/contexts/SoundNotificationContext";
import { toast } from "sonner";

export default function TestNotificationsPage() {
  const { playNotification, isEnabled } = useSoundNotification();

  const testNotifications = [
    {
      type: 'message',
      title: 'Nova Mensagem',
      description: 'Teste de mensagem do WhatsApp',
      action: () => {
        playNotification();
        toast.info('Nova mensagem recebida', {
          description: 'De: Cliente Teste'
        });
      }
    },
    {
      type: 'order',
      title: 'Pedido Atualizado',
      description: 'Teste de atualização de pedido',
      action: () => {
        playNotification();
        toast.success('Pedido atualizado', {
          description: 'Pedido #12345 - Enviado'
        });
      }
    },
    {
      type: 'general',
      title: 'Notificação Geral',
      description: 'Teste de notificação geral',
      action: () => {
        playNotification();
        toast.info('Notificação', {
          description: 'Esta é uma notificação de teste'
        });
      }
    }
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Teste de Notificações</h1>
        <p className="text-muted-foreground">Teste os alertas sonoros e notificações do sistema</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Status dos Alertas Sonoros</CardTitle>
          <CardDescription>
            Alertas sonoros estão {isEnabled ? 'habilitados' : 'desabilitados'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {testNotifications.map((notification, index) => (
              <Card key={index} className="border-2">
                <CardHeader>
                  <CardTitle className="text-lg">{notification.title}</CardTitle>
                  <CardDescription>{notification.description}</CardDescription>
                </CardHeader>
                <CardContent>
                  <Button 
                    onClick={notification.action}
                    className="w-full"
                    variant={index === 0 ? "default" : index === 1 ? "secondary" : "outline"}
                  >
                    Testar {notification.type}
                  </Button>
                </CardContent>
              </Card>
            ))}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Como funciona</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="text-sm text-muted-foreground space-y-2">
            <p>• <strong>Alertas Sonoros:</strong> Podem ser habilitados/desabilitados pelo botão no header</p>
            <p>• <strong>WebSocket:</strong> Conexão automática ao carregar o sistema</p>
            <p>• <strong>Notificações:</strong> Aparecem automaticamente quando webhooks são recebidos</p>
            <p>• <strong>Som:</strong> Reproduzido apenas se os alertas estiverem habilitados</p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
