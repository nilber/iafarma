import React, { createContext, useContext, useState, useRef, useEffect } from 'react';

interface SoundNotificationContextType {
  isEnabled: boolean;
  toggleSound: () => void;
  playNotification: () => void;
  playHumanSupportAlert: () => void;
}

const SoundNotificationContext = createContext<SoundNotificationContextType | undefined>(undefined);

export function SoundNotificationProvider({ children }: { children: React.ReactNode }) {
  const [isEnabled, setIsEnabled] = useState(() => {
    const saved = localStorage.getItem('soundNotifications');
    return saved !== null ? JSON.parse(saved) : true;
  });
  
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const humanSupportAudioRef = useRef<HTMLAudioElement | null>(null);

  useEffect(() => {
    // Criar o elemento de áudio para notificações
    audioRef.current = new Audio();
    // Usando um som de notificação simples (data URL)
    audioRef.current.src = "data:audio/wav;base64,UklGRnoGAABXQVZFZm10IBAAAAABAAEAQB8AAEAfAAABAAgAZGF0YQoGAACBhYqFbF1fdJivrJBhNjVgodDbq2EcBj+a2/LDciUFLIHO8tiJNwgZaLvt559NEAxQp+PwtmMcBjiR1/LMeSwFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSs=";
    audioRef.current.volume = 0.5;
    
    // Criar o elemento de áudio para alertas de atendimento humano (som diferenciado)
    humanSupportAudioRef.current = new Audio();
    humanSupportAudioRef.current.src = "data:audio/wav;base64,UklGRnoGAABXQVZFZm10IBAAAAABAAEAQB8AAEAfAAABAAgAZGF0YQoGAACBhYqFbF1fdJivrJBhNjVgodDbq2EcBj+a2/LDciUFLIHO8tiJNwgZaLvt559NEAxQp+PwtmMcBjiR1/LMeSwFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSsFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwfBDuN1vLNeSs=";
    humanSupportAudioRef.current.volume = 0.7; // Volume mais alto para atendimento humano
    
    return () => {
      if (audioRef.current) {
        audioRef.current = null;
      }
      if (humanSupportAudioRef.current) {
        humanSupportAudioRef.current = null;
      }
    };
  }, []);

  useEffect(() => {
    localStorage.setItem('soundNotifications', JSON.stringify(isEnabled));
  }, [isEnabled]);

  const toggleSound = () => {
    setIsEnabled(!isEnabled);
  };

  const playNotification = () => {
    if (isEnabled && audioRef.current) {
      audioRef.current.currentTime = 0;
      audioRef.current.play().catch(console.error);
    }
  };

  const playHumanSupportAlert = () => {
    if (isEnabled && humanSupportAudioRef.current) {
      // Toca 3 vezes seguidas para chamar mais atenção
      humanSupportAudioRef.current.currentTime = 0;
      humanSupportAudioRef.current.play().catch(console.error);
      
      setTimeout(() => {
        if (humanSupportAudioRef.current) {
          humanSupportAudioRef.current.currentTime = 0;
          humanSupportAudioRef.current.play().catch(console.error);
        }
      }, 500);
      
      setTimeout(() => {
        if (humanSupportAudioRef.current) {
          humanSupportAudioRef.current.currentTime = 0;
          humanSupportAudioRef.current.play().catch(console.error);
        }
      }, 1000);
    }
  };

  return (
    <SoundNotificationContext.Provider value={{ isEnabled, toggleSound, playNotification, playHumanSupportAlert }}>
      {children}
    </SoundNotificationContext.Provider>
  );
}

export function useSoundNotification() {
  const context = useContext(SoundNotificationContext);
  if (context === undefined) {
    throw new Error('useSoundNotification must be used within a SoundNotificationProvider');
  }
  return context;
}
