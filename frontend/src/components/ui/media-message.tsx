import { useState } from 'react';
import { FileIcon, DownloadIcon, PlayIcon, VolumeXIcon, Volume2Icon } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogTrigger } from '@/components/ui/dialog';
import { cn } from '@/lib/utils';

interface MediaMessageProps {
  type?: string;
  mediaUrl?: string;
  mediaType?: string;
  filename?: string;
  content?: string;
  className?: string;
}

export function MediaMessage({ 
  type, 
  mediaUrl, 
  mediaType, 
  filename, 
  content, 
  className 
}: MediaMessageProps) {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isAudioPlaying, setIsAudioPlaying] = useState(false);
  const [isMuted, setIsMuted] = useState(false);

  // If it's a text message or no media, return null to fallback to text rendering
  if (!type || type === 'text' || !mediaUrl) {
    return null;
  }

  const handleDownload = (e?: React.MouseEvent) => {
    if (e) {
      e.preventDefault();
      e.stopPropagation();
    }
    
    if (mediaUrl) {
      // Para PDFs e documentos, abrir em nova aba primeiro
      if (mediaType?.includes('pdf') || mediaType?.includes('document')) {
        window.open(mediaUrl, '_blank', 'noopener,noreferrer');
        return;
      }
      
      // Para outros arquivos, fazer download direto
      try {
        const link = document.createElement('a');
        link.href = mediaUrl;
        link.download = filename || `file.${getFileExtension(mediaType)}`;
        link.target = '_blank';
        link.rel = 'noopener noreferrer';
        
        // Adicionar temporariamente ao DOM
        document.body.appendChild(link);
        link.click();
        
        // Remover após um pequeno delay
        setTimeout(() => {
          document.body.removeChild(link);
        }, 100);
      } catch (error) {
        console.error('Erro no download:', error);
        // Fallback: abrir em nova aba
        window.open(mediaUrl, '_blank', 'noopener,noreferrer');
      }
    }
  };

  const getFileExtension = (mimeType?: string) => {
    if (!mimeType) return 'file';
    const extensions: { [key: string]: string } = {
      'image/jpeg': 'jpg',
      'image/png': 'png',
      'image/gif': 'gif',
      'video/mp4': 'mp4',
      'audio/ogg': 'ogg',
      'audio/mpeg': 'mp3',
      'application/pdf': 'pdf',
      'application/msword': 'doc',
      'application/vnd.openxmlformats-officedocument.wordprocessingml.document': 'docx'
    };
    return extensions[mimeType] || 'file';
  };

  const getFileIcon = () => {
    if (!mediaType) return <FileIcon className="w-8 h-8" />;
    
    if (mediaType.startsWith('image/')) {
      return <div className="w-8 h-8 bg-blue-100 rounded flex items-center justify-center text-blue-600 text-xs font-bold">IMG</div>;
    }
    if (mediaType.startsWith('video/')) {
      return <div className="w-8 h-8 bg-red-100 rounded flex items-center justify-center text-red-600 text-xs font-bold">VID</div>;
    }
    if (mediaType.startsWith('audio/')) {
      return <div className="w-8 h-8 bg-green-100 rounded flex items-center justify-center text-green-600 text-xs font-bold">AUD</div>;
    }
    if (mediaType === 'application/pdf') {
      return <div className="w-8 h-8 bg-orange-100 rounded flex items-center justify-center text-orange-600 text-xs font-bold">PDF</div>;
    }
    
    return <FileIcon className="w-8 h-8" />;
  };

  // Image rendering
  if (type === 'image' && mediaType?.startsWith('image/')) {
    return (
      <div className={cn("max-w-xs", className)}>
        <Dialog open={isModalOpen} onOpenChange={setIsModalOpen}>
          <DialogTrigger asChild>
            <div className="cursor-pointer group relative">
              <img 
                src={mediaUrl} 
                alt={filename || 'Imagem'} 
                className="rounded-lg max-w-full h-auto hover:opacity-90 transition-opacity"
                style={{ maxHeight: '200px' }}
              />
              <div className="absolute inset-0 bg-black bg-opacity-0 group-hover:bg-opacity-10 transition-all rounded-lg flex items-center justify-center">
                <div className="opacity-0 group-hover:opacity-100 transition-opacity text-white text-sm bg-black bg-opacity-50 px-2 py-1 rounded">
                  Clique para ampliar
                </div>
              </div>
            </div>
          </DialogTrigger>
          <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto p-0">
            <div className="relative">
              <img 
                src={mediaUrl} 
                alt={filename || 'Imagem'} 
                className="w-full h-auto max-h-[80vh] object-contain"
              />
              <Button
                onClick={(e) => handleDownload(e)}
                className="absolute top-4 right-4 bg-black bg-opacity-50 hover:bg-opacity-70"
                size="sm"
              >
                <DownloadIcon className="w-4 h-4 mr-2" />
                Download
              </Button>
            </div>
          </DialogContent>
        </Dialog>
        {content && (
          <p className="text-sm mt-2 text-gray-600">{content}</p>
        )}
      </div>
    );
  }

  // Video rendering
  if (type === 'video' && mediaType?.startsWith('video/')) {
    return (
      <div className={cn("max-w-xs", className)}>
        <Dialog open={isModalOpen} onOpenChange={setIsModalOpen}>
          <DialogTrigger asChild>
            <div className="cursor-pointer group relative">
              <video 
                className="rounded-lg max-w-full h-auto"
                style={{ maxHeight: '200px' }}
                poster={mediaUrl} // You might want to add a thumbnail endpoint
              >
                <source src={mediaUrl} type={mediaType} />
                Seu navegador não suporta vídeos.
              </video>
              <div className="absolute inset-0 bg-black bg-opacity-20 group-hover:bg-opacity-30 transition-all rounded-lg flex items-center justify-center">
                <PlayIcon className="w-12 h-12 text-white opacity-80" />
              </div>
            </div>
          </DialogTrigger>
          <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto p-4">
            <div className="relative">
              <video 
                controls 
                className="w-full h-auto max-h-[70vh]"
                autoPlay
              >
                <source src={mediaUrl} type={mediaType} />
                Seu navegador não suporta vídeos.
              </video>
              <Button
                onClick={(e) => handleDownload(e)}
                className="absolute top-4 right-4 bg-black bg-opacity-50 hover:bg-opacity-70"
                size="sm"
              >
                <DownloadIcon className="w-4 h-4 mr-2" />
                Download
              </Button>
            </div>
          </DialogContent>
        </Dialog>
        {content && (
          <p className="text-sm mt-2 text-gray-600">{content}</p>
        )}
      </div>
    );
  }

  // Audio rendering
  if (type === 'audio' && mediaType?.startsWith('audio/')) {
    return (
      <div className={cn("max-w-sm", className)} style={{ minWidth: '400px' }}>
        <div className="bg-gray-100 rounded-lg p-4">
          <div className="flex items-center space-x-3 mb-3">
            <div className="w-8 h-8 bg-green-100 rounded-full flex items-center justify-center">
              <Volume2Icon className="w-4 h-4 text-green-600" />
            </div>
            <div className="flex-1">
              <p className="text-sm font-medium text-gray-700">
                {filename || 'Áudio'}
              </p>
            </div>
          </div>
          <audio controls className="w-full h-8" style={{ minHeight: '32px' }}>
            <source src={mediaUrl} type={mediaType} />
            Seu navegador não suporta áudio.
          </audio>
        </div>
        {content && (
          <p className="text-sm mt-2 text-gray-600">{content}</p>
        )}
      </div>
    );
  }

  // Document and other file types
  return (
    <div className={cn("max-w-xs", className)}>
      <div className="bg-gray-100 rounded-lg p-3 flex items-center space-x-3 hover:bg-gray-200 transition-colors">
        {getFileIcon()}
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-gray-900 truncate">
            {filename || `Arquivo ${type}`}
          </p>
          {mediaType && (
            <p className="text-xs text-gray-500">{mediaType}</p>
          )}
        </div>
        <Button
          onClick={(e) => handleDownload(e)}
          variant="ghost"
          size="sm"
          className="p-2 hover:bg-gray-300"
        >
          <DownloadIcon className="w-4 h-4" />
        </Button>
      </div>
      {content && (
        <p className="text-sm mt-2 text-gray-600">{content}</p>
      )}
    </div>
  );
}
