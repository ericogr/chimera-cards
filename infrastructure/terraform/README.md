# Quimera - Terraform (OCI)

Este diretório contém a automação Terraform usada para provisionar a
infraestrutura Oracle Cloud (VCN, subnets, instâncias e Load Balancer).

## Conteúdo

- [`main.tf`](main.tf) — VCN, Subnet, Internet Gateway, Route Table, Security List, Compute Instance, Load Balancer
- [`variables.tf`](variables.tf) — Variáveis parametrizáveis
- [`outputs.tf`](outputs.tf) — Endereços IP públicos (instância e LB)
- [`versions.tf`](versions.tf) — Provider e versão do Terraform
- [`terraform.tfvars.example`](terraform.tfvars.example) — Exemplo de variáveis

## Como usar (resumo)

1. Copie o arquivo de exemplo e edite os valores necessários (pelo menos
   `compartment_ocid` e `ssh_public_key`):

```bash
cp terraform.tfvars.example terraform.tfvars
# edite terraform.tfvars
```

2. Ajuste `image_ocid` caso possua o OCID da imagem; caso contrário o
   código tenta localizar pela `image_display_name` (pode falhar dependendo
   da tenancy/região).

3. Inicialize e aplique:

```bash
terraform init
terraform plan
terraform apply
```

## Observações / suposições

- O `ssh_public_key` deve conter o conteúdo do seu `~/.ssh/id_rsa.pub` (formato OpenSSH).
- A busca automática de imagem usa `image_display_name` como fallback; quando possível prefira informar `image_ocid`.
- Para configurar OCPUs/memória, use um shape flexível (ex.: `VM.Standard.E3.Flex`) e defina `use_flex_shape = true`.
- `storage_size_gb` define o tamanho do boot volume (substitui o tamanho padrão da imagem).
- O Load Balancer usa TCP e encaminha conexões da porta `lb_listen_port` (padrão 443) para `backend_port` na instância.
