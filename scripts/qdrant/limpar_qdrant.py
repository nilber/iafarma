#!/usr/bin/env python3
"""
Script para limpar dados do Qdrant
Permite limpar por tenant específico ou todas as coleções
"""
import os
import sys
import argparse
from qdrant_client import QdrantClient
from qdrant_client.http import models
from qdrant_client.http.exceptions import ResponseHandlingException

# Configurações padrão (podem ser sobrescritas por variáveis de ambiente)
QDRANT_URL = os.getenv('QDRANT_URL', 'localhost:6334')
QDRANT_PASSWORD = os.getenv('QDRANT_PASSWORD', '123456')

def get_qdrant_client():
    """Cria conexão com o Qdrant"""
    try:
        # Separar host e porta
        if ':' in QDRANT_URL:
            host, port = QDRANT_URL.split(':')
        else:
            host = QDRANT_URL
            port = '6333'  # HTTP port
        
        # Tentar primeira conexão HTTP
        try:
            client = QdrantClient(
                host=host,
                port=6333,  # HTTP port
                api_key=QDRANT_PASSWORD if QDRANT_PASSWORD else None,
                prefer_grpc=False
            )
            collections = client.get_collections()
            print(f"✅ Conectado ao Qdrant em {host}:6333 (HTTP)")
            return client
        except:
            # Fallback para gRPC sem SSL
            client = QdrantClient(
                host=host,
                port=int(port),
                api_key=QDRANT_PASSWORD if QDRANT_PASSWORD else None,
                prefer_grpc=True,
                https=False
            )
            collections = client.get_collections()
            print(f"✅ Conectado ao Qdrant em {QDRANT_URL} (gRPC)")
            return client
    except Exception as e:
        print(f"❌ Erro ao conectar no Qdrant: {e}")
        print("💡 Verifique se o Qdrant está rodando e as configurações estão corretas")
        print("💡 Configurações atuais:")
        print(f"   QDRANT_URL: {QDRANT_URL}")
        print(f"   QDRANT_PASSWORD: {'***' if QDRANT_PASSWORD else 'não definida'}")
        return None

def list_collections(client):
    """Lista todas as coleções existentes"""
    try:
        collections = client.get_collections()
        if not collections.collections:
            print("📭 Nenhuma coleção encontrada")
            return []
        
        collection_names = [col.name for col in collections.collections]
        print(f"📦 Coleções encontradas ({len(collection_names)}):")
        
        for name in collection_names:
            try:
                info = client.get_collection(name)
                point_count = info.points_count if hasattr(info, 'points_count') else 0
                print(f"   📂 {name} ({point_count} pontos)")
            except:
                print(f"   📂 {name} (erro ao obter info)")
        
        return collection_names
    except Exception as e:
        print(f"❌ Erro ao listar coleções: {e}")
        return []

def get_collection_info(client, collection_name):
    """Obtém informações detalhadas de uma coleção"""
    try:
        info = client.get_collection(collection_name)
        point_count = info.points_count if hasattr(info, 'points_count') else 0
        return point_count
    except:
        return 0

def delete_collection(client, collection_name, confirm=True):
    """Deleta uma coleção específica"""
    try:
        if confirm:
            response = input(f"⚠️ Tem certeza que deseja deletar a coleção '{collection_name}'? (s/N): ")
            if response.lower() not in ['s', 'sim', 'y', 'yes']:
                print("❌ Operação cancelada")
                return False
        
        print(f"🗑️ Deletando coleção '{collection_name}'...")
        client.delete_collection(collection_name)
        print(f"✅ Coleção '{collection_name}' deletada com sucesso!")
        return True
    except Exception as e:
        print(f"❌ Erro ao deletar coleção '{collection_name}': {e}")
        return False

def clear_collection_points(client, collection_name, confirm=True):
    """Remove todos os pontos de uma coleção sem deletar a coleção"""
    try:
        if confirm:
            response = input(f"⚠️ Tem certeza que deseja limpar todos os pontos da coleção '{collection_name}'? (s/N): ")
            if response.lower() not in ['s', 'sim', 'y', 'yes']:
                print("❌ Operação cancelada")
                return False
        
        point_count = get_collection_info(client, collection_name)
        
        if point_count == 0:
            print(f"📭 A coleção '{collection_name}' já está vazia")
            return True
        
        print(f"🧹 Limpando {point_count} pontos da coleção '{collection_name}'...")
        
        # Deletar todos os pontos usando filtro vazio
        client.delete(
            collection_name=collection_name,
            points_selector=models.FilterSelector(
                filter=models.Filter()
            )
        )
        
        print(f"✅ Pontos removidos da coleção '{collection_name}'!")
        return True
    except Exception as e:
        print(f"❌ Erro ao limpar pontos da coleção '{collection_name}': {e}")
        return False

def clean_tenant_collections(client, tenant_id, action='clear'):
    """Limpa coleções de um tenant específico"""
    collection_names = list_collections(client)
    if not collection_names:
        return
    
    # Padrões de coleções do sistema
    tenant_patterns = [
        f"products_{tenant_id}",
        f"conversations_{tenant_id}"
    ]
    
    found_collections = []
    for pattern in tenant_patterns:
        if pattern in collection_names:
            found_collections.append(pattern)
    
    if not found_collections:
        print(f"📭 Nenhuma coleção encontrada para o tenant '{tenant_id}'")
        return
    
    print(f"🎯 Coleções encontradas para tenant '{tenant_id}':")
    for col in found_collections:
        point_count = get_collection_info(client, col)
        print(f"   📂 {col} ({point_count} pontos)")
    
    confirm = input(f"\n⚠️ Tem certeza que deseja {'deletar' if action == 'delete' else 'limpar'} essas coleções? (s/N): ")
    if confirm.lower() not in ['s', 'sim', 'y', 'yes']:
        print("❌ Operação cancelada")
        return
    
    success_count = 0
    for col_name in found_collections:
        if action == 'delete':
            if delete_collection(client, col_name, confirm=False):
                success_count += 1
        else:
            if clear_collection_points(client, col_name, confirm=False):
                success_count += 1
    
    print(f"\n✅ {success_count}/{len(found_collections)} coleções processadas com sucesso!")

def clean_all_collections(client, action='clear'):
    """Limpa todas as coleções"""
    collection_names = list_collections(client)
    if not collection_names:
        return
    
    total_points = 0
    for col_name in collection_names:
        point_count = get_collection_info(client, col_name)
        total_points += point_count
    
    print(f"\n📊 Total: {len(collection_names)} coleções com {total_points} pontos")
    
    confirm = input(f"\n⚠️ ATENÇÃO: Tem certeza que deseja {'DELETAR TODAS' if action == 'delete' else 'LIMPAR TODAS'} as coleções? (s/N): ")
    if confirm.lower() not in ['s', 'sim', 'y', 'yes']:
        print("❌ Operação cancelada")
        return
    
    success_count = 0
    for col_name in collection_names:
        print(f"\n🔄 Processando '{col_name}'...")
        if action == 'delete':
            if delete_collection(client, col_name, confirm=False):
                success_count += 1
        else:
            if clear_collection_points(client, col_name, confirm=False):
                success_count += 1
    
    print(f"\n✅ {success_count}/{len(collection_names)} coleções processadas com sucesso!")

def main():
    parser = argparse.ArgumentParser(
        description='Script para limpar dados do Qdrant',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Exemplos de uso:
  python3 limpar_qdrant.py --list                     # Listar coleções
  python3 limpar_qdrant.py --tenant abc123 --clear    # Limpar pontos do tenant
  python3 limpar_qdrant.py --tenant abc123 --delete   # Deletar coleções do tenant
  python3 limpar_qdrant.py --all --clear              # Limpar pontos de todas coleções
  python3 limpar_qdrant.py --all --delete             # Deletar todas as coleções
  
Configuração:
  QDRANT_URL=localhost:6334 (padrão)
  QDRANT_PASSWORD=123456 (padrão)
        """
    )
    
    parser.add_argument('--list', action='store_true', help='Listar todas as coleções')
    parser.add_argument('--tenant', type=str, help='ID do tenant para operação específica')
    parser.add_argument('--all', action='store_true', help='Processar todas as coleções')
    parser.add_argument('--clear', action='store_true', help='Limpar pontos (manter estrutura da coleção)')
    parser.add_argument('--delete', action='store_true', help='Deletar coleções completamente')
    parser.add_argument('--url', type=str, help='URL do Qdrant (sobrescreve QDRANT_URL)')
    parser.add_argument('--password', type=str, help='Senha do Qdrant (sobrescreve QDRANT_PASSWORD)')
    
    args = parser.parse_args()
    
    # Sobrescrever configurações se fornecidas
    global QDRANT_URL, QDRANT_PASSWORD
    if args.url:
        QDRANT_URL = args.url
    if args.password:
        QDRANT_PASSWORD = args.password
    
    print(f"🚀 Script de Limpeza do Qdrant")
    print(f"🔗 Conectando em: {QDRANT_URL}")
    print("=" * 50)
    
    # Conectar ao Qdrant
    client = get_qdrant_client()
    if not client:
        sys.exit(1)
    
    # Verificar argumentos
    if args.list:
        list_collections(client)
        return
    
    if not args.clear and not args.delete:
        print("❌ Especifique --clear ou --delete")
        parser.print_help()
        return
    
    if args.clear and args.delete:
        print("❌ Use apenas --clear OU --delete, não ambos")
        return
    
    action = 'delete' if args.delete else 'clear'
    
    if args.tenant:
        clean_tenant_collections(client, args.tenant, action)
    elif args.all:
        clean_all_collections(client, action)
    else:
        print("❌ Especifique --tenant <ID> ou --all")
        parser.print_help()

if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\n❌ Operação interrompida pelo usuário")
        sys.exit(1)
    except Exception as e:
        print(f"\n❌ Erro inesperado: {e}")
        sys.exit(1)