import { useState, useRef } from "react";
import { Upload, X, Loader2, ImageIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { useProductImages, useUploadProductImage, useDeleteProductImage } from "@/lib/api/hooks";
import { toast } from "sonner";

interface ProductImagesProps {
  productId: string;
}

export default function ProductImages({ productId }: ProductImagesProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [uploadingFiles, setUploadingFiles] = useState<Set<string>>(new Set());

  const { data: images = [], isLoading, refetch } = useProductImages(productId);
  const uploadImageMutation = useUploadProductImage();
  const deleteImageMutation = useDeleteProductImage();

  const handleFileSelect = () => {
    fileInputRef.current?.click();
  };

  const handleFileChange = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files || []);
    
    for (const file of files) {
      if (!file.type.startsWith('image/')) {
        toast.error(`${file.name} não é uma imagem válida`);
        continue;
      }

      if (file.size > 5 * 1024 * 1024) { // 5MB limit
        toast.error(`${file.name} é muito grande (máximo 5MB)`);
        continue;
      }

      const fileId = `${file.name}-${Date.now()}`;
      setUploadingFiles(prev => new Set(prev).add(fileId));

      try {
        await uploadImageMutation.mutateAsync({
          productId,
          file,
        });
        
        toast.success(`${file.name} enviada com sucesso!`);
        refetch();
      } catch (error) {
        console.error("Erro ao enviar imagem:", error);
        toast.error(`Erro ao enviar ${file.name}`);
      } finally {
        setUploadingFiles(prev => {
          const newSet = new Set(prev);
          newSet.delete(fileId);
          return newSet;
        });
      }
    }

    // Reset file input
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const handleDeleteImage = async (imageId: string) => {
    if (!confirm('Tem certeza que deseja excluir esta imagem?')) {
      return;
    }

    try {
      await deleteImageMutation.mutateAsync({ productId, imageId });
      refetch();
      toast.success('Imagem excluída com sucesso!');
    } catch (error) {
      console.error("Erro ao excluir imagem:", error);
      toast.error('Erro ao excluir imagem');
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="w-6 h-6 animate-spin" />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium">Imagens do Produto</h3>
        <Button onClick={handleFileSelect} disabled={uploadImageMutation.isPending}>
          <Upload className="w-4 h-4 mr-2" />
          Adicionar Imagens
        </Button>
      </div>

      <input
        ref={fileInputRef}
        type="file"
        accept="image/*"
        multiple
        onChange={handleFileChange}
        className="hidden"
      />

      {images.length === 0 && uploadingFiles.size === 0 ? (
        <Card>
          <CardContent className="text-center py-8">
            <ImageIcon className="w-12 h-12 text-muted-foreground mx-auto mb-4" />
            <p className="text-muted-foreground mb-4">
              Nenhuma imagem adicionada ainda
            </p>
            <Button onClick={handleFileSelect}>
              <Upload className="w-4 h-4 mr-2" />
              Adicionar Primeira Imagem
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
          {/* Existing images */}
          {images.map((image) => (
            <Card key={image.id} className="relative group">
              <CardContent className="p-2">
                <div className="aspect-square relative rounded-lg overflow-hidden bg-muted">
                  <img
                    src={image.url}
                    alt={image.alt || "Imagem do produto"}
                    className="w-full h-full object-cover"
                    onError={(e) => {
                      const target = e.target as HTMLImageElement;
                      target.src = 'data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTIxIDEySC0zTTIxIDEySC0zIiBzdHJva2U9IiNjY2MiIHN0cm9rZS13aWR0aD0iMiIvPgo8L3N2Zz4K';
                    }}
                  />
                  <Button
                    variant="destructive"
                    size="sm"
                    className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity"
                    onClick={() => handleDeleteImage(image.id)}
                    disabled={deleteImageMutation.isPending}
                  >
                    <X className="w-3 h-3" />
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}

          {/* Uploading placeholders */}
          {Array.from(uploadingFiles).map((fileId) => (
            <Card key={fileId}>
              <CardContent className="p-2">
                <div className="aspect-square flex items-center justify-center bg-muted rounded-lg">
                  <div className="text-center">
                    <Loader2 className="w-6 h-6 animate-spin mx-auto mb-2" />
                    <p className="text-xs text-muted-foreground">Enviando...</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}

          {/* Upload area */}
          <Card 
            className="border-dashed border-2 hover:border-primary/50 transition-colors cursor-pointer"
            onClick={handleFileSelect}
          >
            <CardContent className="p-2">
              <div className="aspect-square flex items-center justify-center">
                <div className="text-center">
                  <Upload className="w-6 h-6 mx-auto mb-2 text-muted-foreground" />
                  <p className="text-xs text-muted-foreground">Adicionar</p>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {uploadImageMutation.isPending && (
        <div className="text-center py-4">
          <div className="flex items-center justify-center gap-2 text-muted-foreground">
            <Loader2 className="w-4 h-4 animate-spin" />
            <span className="text-sm">Enviando imagens...</span>
          </div>
        </div>
      )}
    </div>
  );
}