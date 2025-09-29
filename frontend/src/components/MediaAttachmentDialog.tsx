import { useState, useRef, useCallback } from 'react';
import { Paperclip, Image, FileText, Mic, X, Play, Pause, Send, Loader2, Eye } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { toast } from 'sonner';
import { apiClient } from '@/lib/api/client';
import { useSendWhatsAppImage, useSendWhatsAppDocument, useSendWhatsAppAudio } from '@/lib/api/hooks';

interface MediaAttachmentDialogProps {
  isOpen: boolean;
  onClose: () => void;
  conversationId: string;
  chatId: string;
}

export interface MediaAttachmentData {
  type: 'image' | 'document' | 'audio';
  file: File;
  caption?: string;
  url?: string;
}

export function MediaAttachmentDialog({ isOpen, onClose, conversationId, chatId }: MediaAttachmentDialogProps) {
  const [selectedType, setSelectedType] = useState<'image' | 'document' | 'audio' | null>(null);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [caption, setCaption] = useState('');
  const [isRecording, setIsRecording] = useState(false);
  const [audioUrl, setAudioUrl] = useState<string | null>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  
  const fileInputRef = useRef<HTMLInputElement>(null);
  const mediaRecorderRef = useRef<MediaRecorder | null>(null);
  const audioChunksRef = useRef<Blob[]>([]);
  const audioRef = useRef<HTMLAudioElement>(null);

  // Hooks para envio
  const sendImage = useSendWhatsAppImage();
  const sendDocument = useSendWhatsAppDocument();
  const sendAudio = useSendWhatsAppAudio();

  const resetState = useCallback(() => {
    setSelectedType(null);
    setSelectedFile(null);
    setCaption('');
    setIsRecording(false);
    setAudioUrl(null);
    setIsPlaying(false);
    setPreviewUrl(null);
    audioChunksRef.current = [];
  }, []);

  const handleClose = useCallback(() => {
    if (audioUrl) {
      URL.revokeObjectURL(audioUrl);
    }
    if (previewUrl) {
      URL.revokeObjectURL(previewUrl);
    }
    resetState();
    onClose();
  }, [audioUrl, previewUrl, resetState, onClose]);

  const handleTypeSelect = (type: 'image' | 'document' | 'audio') => {
    setSelectedType(type);
    
    if (type === 'audio') {
      // Start audio recording immediately
      startRecording();
    } else {
      // Open file picker for images and documents
      const input = fileInputRef.current;
      if (input) {
        input.accept = type === 'image' ? 'image/*' : '*/*';
        input.click();
      }
    }
  };

  const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    setSelectedFile(file);

    // Create preview for images
    if (selectedType === 'image' && file.type.startsWith('image/')) {
      const url = URL.createObjectURL(file);
      setPreviewUrl(url);
    }
  };

  const startRecording = async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      const mediaRecorder = new MediaRecorder(stream, {
        mimeType: 'audio/webm;codecs=opus'
      });
      
      mediaRecorderRef.current = mediaRecorder;
      audioChunksRef.current = [];

      mediaRecorder.ondataavailable = (event) => {
        if (event.data.size > 0) {
          audioChunksRef.current.push(event.data);
        }
      };

      mediaRecorder.onstop = () => {
        const audioBlob = new Blob(audioChunksRef.current, { type: 'audio/webm;codecs=opus' });
        const audioFile = new File([audioBlob], 'audio-message.webm', { type: 'audio/webm;codecs=opus' });
        setSelectedFile(audioFile);
        
        const url = URL.createObjectURL(audioBlob);
        setAudioUrl(url);
        
        // Stop all tracks
        stream.getTracks().forEach(track => track.stop());
      };

      mediaRecorder.start();
      setIsRecording(true);
    } catch (error) {
      console.error('Error starting recording:', error);
      toast.error('Erro ao iniciar gravação. Verifique as permissões de microfone.');
    }
  };

  const stopRecording = () => {
    if (mediaRecorderRef.current && isRecording) {
      mediaRecorderRef.current.stop();
      setIsRecording(false);
    }
  };

  const togglePlayback = () => {
    if (!audioRef.current || !audioUrl) return;

    if (isPlaying) {
      audioRef.current.pause();
      setIsPlaying(false);
    } else {
      audioRef.current.play();
      setIsPlaying(true);
    }
  };

  const uploadFileToS3 = async (file: File): Promise<{ url: string; messageId: string }> => {
    const formData = new FormData();
    formData.append('file', file);

    const response = await apiClient.uploadMedia(formData, selectedType || 'document');
    return response;
  };

  const handleSend = async () => {
    if (!selectedFile || !selectedType) return;

    setIsUploading(true);
    try {
      // Upload file to S3
      const uploadResult = await uploadFileToS3(selectedFile);
      
      // Prepare file data
      const fileData = {
        mimetype: selectedFile.type,
        filename: selectedFile.name,
        url: uploadResult.url,
      };

      // Send based on type
      let response;
      if (selectedType === 'image') {
        response = await sendImage.mutateAsync({
          chatId,
          file: fileData,
          caption: caption.trim() || undefined,
          conversation_id: conversationId,
        });
      } else if (selectedType === 'document') {
        response = await sendDocument.mutateAsync({
          chatId,
          file: fileData,
          caption: caption.trim() || undefined,
          conversation_id: conversationId,
        });
      } else if (selectedType === 'audio') {
        // Convert audio to opus format if needed
        const audioFileData = {
          mimetype: 'audio/ogg; codecs=opus',
          url: uploadResult.url,
        };
        response = await sendAudio.mutateAsync({
          chatId,
          file: audioFileData,
          convert: true,
          conversation_id: conversationId,
        });
      }

      toast.success('Mídia enviada com sucesso!');
      handleClose();
    } catch (error) {
      console.error('Error sending media:', error);
      toast.error('Erro ao enviar mídia. Tente novamente.');
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <>
      <Dialog open={isOpen} onOpenChange={handleClose}>
        <DialogContent className="max-w-md max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Enviar Anexo</DialogTitle>
          </DialogHeader>

          {!selectedType ? (
            <div className="space-y-3">
              <Button
                onClick={() => handleTypeSelect('image')}
                variant="outline"
                className="w-full justify-start"
              >
                <Image className="w-4 h-4 mr-2" />
                Imagem
              </Button>
              <Button
                onClick={() => handleTypeSelect('document')}
                variant="outline"
                className="w-full justify-start"
              >
                <FileText className="w-4 h-4 mr-2" />
                Documento
              </Button>
              <Button
                onClick={() => handleTypeSelect('audio')}
                variant="outline"
                className="w-full justify-start"
              >
                <Mic className="w-4 h-4 mr-2" />
                Áudio
              </Button>
            </div>
          ) : selectedType === 'audio' && isRecording ? (
            <div className="space-y-4 text-center">
              <div className="flex items-center justify-center">
                <div className="w-16 h-16 bg-red-500 rounded-full flex items-center justify-center animate-pulse">
                  <Mic className="w-8 h-8 text-white" />
                </div>
              </div>
              <p className="text-sm text-muted-foreground">Gravando áudio...</p>
              <Button onClick={stopRecording} variant="outline">
                Parar Gravação
              </Button>
            </div>
          ) : selectedFile ? (
            <div className="space-y-4">
              {/* Preview Area */}
              <div className="border-2 border-dashed border-gray-200 rounded-lg p-4">
                {selectedType === 'image' && previewUrl ? (
                  <div className="relative">
                    <img 
                      src={previewUrl} 
                      alt="Preview" 
                      className="w-full max-h-48 object-contain rounded"
                    />
                    <Button
                      onClick={() => window.open(previewUrl, '_blank')}
                      size="sm"
                      variant="secondary"
                      className="absolute top-2 right-2"
                    >
                      <Eye className="w-4 h-4" />
                    </Button>
                  </div>
                ) : selectedType === 'audio' && audioUrl ? (
                  <div className="flex items-center justify-center space-x-4">
                    <Button
                      onClick={togglePlayback}
                      size="sm"
                      variant="outline"
                    >
                      {isPlaying ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
                    </Button>
                    <span className="text-sm text-muted-foreground">
                      Áudio gravado ({Math.round(selectedFile.size / 1024)}KB)
                    </span>
                    <audio
                      ref={audioRef}
                      src={audioUrl}
                      onEnded={() => setIsPlaying(false)}
                      style={{ display: 'none' }}
                    />
                  </div>
                ) : (
                  <div className="text-center">
                    <FileText className="w-8 h-8 mx-auto text-muted-foreground mb-2" />
                    <p className="text-sm text-muted-foreground">{selectedFile.name}</p>
                    <p className="text-xs text-muted-foreground">
                      {Math.round(selectedFile.size / 1024)}KB
                    </p>
                  </div>
                )}
              </div>

              {/* Caption Input */}
              <div className="space-y-2">
                <Label htmlFor="caption">Legenda (opcional)</Label>
                <Textarea
                  id="caption"
                  placeholder="Adicione uma legenda..."
                  value={caption}
                  onChange={(e) => setCaption(e.target.value)}
                  rows={3}
                />
              </div>

              {/* Action Buttons */}
              <div className="flex justify-between space-x-2">
                <Button onClick={handleClose} variant="outline" disabled={isUploading}>
                  Cancelar
                </Button>
                {selectedType === 'audio' && (
                  <Button 
                    onClick={() => {
                      if (audioUrl) URL.revokeObjectURL(audioUrl);
                      setAudioUrl(null);
                      setSelectedFile(null);
                      startRecording();
                    }} 
                    variant="outline"
                    disabled={isUploading}
                  >
                    Gravar Novamente
                  </Button>
                )}
                <Button 
                  onClick={handleSend} 
                  className="bg-whatsapp hover:bg-whatsapp-dark"
                  disabled={isUploading}
                >
                  {isUploading ? (
                    <Loader2 className="w-4 h-4 animate-spin mr-2" />
                  ) : (
                    <Send className="w-4 h-4 mr-2" />
                  )}
                  Enviar
                </Button>
              </div>
            </div>
          ) : null}
        </DialogContent>
      </Dialog>

      {/* Hidden file input */}
      <input
        ref={fileInputRef}
        type="file"
        onChange={handleFileSelect}
        style={{ display: 'none' }}
      />
    </>
  );
}
