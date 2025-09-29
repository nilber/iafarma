import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Zap, Loader2, Copy, Sparkles } from "lucide-react";
import { apiClient } from "@/lib/api/client";
import { ProductGenerationRequest, ProductGenerationResult } from "@/lib/api/types";
import { toast } from "sonner";

interface AIProductGeneratorProps {
  onProductGenerated: (data: {
    name: string;
    description: string;
    price: string;
    sku?: string;
    brand?: string;
    tags?: string;
  }) => void;
}

export function AIProductGenerator({ onProductGenerated }: AIProductGeneratorProps) {
  const [prompt, setPrompt] = useState("");
  const [generating, setGenerating] = useState(false);
  const [lastGeneration, setLastGeneration] = useState<ProductGenerationResult | null>(null);

  const handleGenerate = async () => {
    if (!prompt.trim()) {
      toast.error("Digite uma descrição do produto para gerar");
      return;
    }

    setGenerating(true);
    try {
      const request: ProductGenerationRequest = {
        product_name: prompt.trim()
      };

      const response = await apiClient.generateProductInfo(request);
      setLastGeneration(response);
      
      toast.success("Produto gerado com IA! Use os dados gerados.");
    } catch (error: any) {
      if (error.message?.includes("créditos insuficientes")) {
        toast.error("Créditos insuficientes para gerar produto com IA");
      } else {
        toast.error("Erro ao gerar produto com IA");
      }
      console.error("Erro na geração:", error);
    } finally {
      setGenerating(false);
    }
  };

  const handleCopyToForm = () => {
    if (!lastGeneration) return;

    const productInfo = lastGeneration.product_info;
    
    onProductGenerated({
      name: prompt, // Use the original prompt as the name
      description: productInfo.description,
      price: "0.00", // Default price since AI doesn't suggest one
      sku: "",
      brand: productInfo.suggestions.brand || "",
      tags: productInfo.tags.join(", "),
    });

    toast.success("Dados copiados para o formulário!");
  };

  return (
    <Card className="mb-6 border-muted bg-muted">
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-muted-foreground">
          <Sparkles className="w-5 h-5" />
          Gerador de Produtos com IA
        </CardTitle>
        <CardDescription>
          Descreva o produto que você quer cadastrar e a IA gerará as informações automaticamente.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="ai-prompt">Descreva o produto</Label>
          <Textarea
            id="ai-prompt"
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="Ex: Smartphone Samsung Galaxy A54 5G, 128GB, câmera tripla, tela 6.4 polegadas"
            rows={3}
            className="resize-none"
          />
        </div>

        <div className="flex gap-2">
          <Button
            type="button"
            onClick={handleGenerate}
            disabled={generating || !prompt.trim()}
            className="flex items-center gap-2"
            variant="outline"
          >
            {generating ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Zap className="w-4 h-4" />
            )}
            {generating ? "Gerando..." : "Gerar com IA"}
          </Button>

          {lastGeneration && (
            <Button
              type="button"
              onClick={handleCopyToForm}
              variant="secondary"
              className="flex items-center gap-2"
            >
              <Copy className="w-4 h-4" />
              Usar dados gerados
            </Button>
          )}
        </div>

        {lastGeneration && (
          <Card className="border-green-200 bg-green-50">
            <CardHeader className="pb-3">
              <CardTitle className="text-sm text-green-700">Produto Gerado</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              <div>
                <strong>Nome:</strong> {prompt}
              </div>
              <div>
                <strong>Descrição:</strong> {lastGeneration.product_info.description}
              </div>
              <div>
                <strong>Tags:</strong> {lastGeneration.product_info.tags.join(", ")}
              </div>
              {lastGeneration.product_info.suggestions.brand && (
                <div>
                  <strong>Marca sugerida:</strong> {lastGeneration.product_info.suggestions.brand}
                </div>
              )}
              {lastGeneration.product_info.suggestions.category && (
                <div>
                  <strong>Categoria sugerida:</strong> {lastGeneration.product_info.suggestions.category}
                </div>
              )}
              <div className="text-xs text-muted-foreground pt-2">
                <strong>Créditos usados:</strong> {lastGeneration.credits_used}
              </div>
            </CardContent>
          </Card>
        )}
      </CardContent>
    </Card>
  );
}
