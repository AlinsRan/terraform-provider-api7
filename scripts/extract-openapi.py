#!/usr/bin/env python3
"""
从 API7 EE 完整 OpenAPI 规范中提取 Terraform Provider 所需的路径子集。

用法：
    python3 scripts/extract-openapi.py \
        --spec <full-spec-path> \
        --out openapi-subset.yaml
"""

import argparse
import sys
import yaml

# Provider 用到的 operation IDs
OPERATION_IDS = {
    # Gateway Group
    "listGatewayGroups",
    "putGatewayGroup",
    # Consumer
    "createConsumer",
    "getConsumer",
    "upsertConsumer",
    "deleteConsumer",
    # Published Service
    "createPublishedService",
    "getPublishedService",
    "putPublishedService",
    "deletePublishedService",
    # Route
    "createPublishedServiceRoute",
    "getPublishedServiceRoute",
    "putPublishedServiceRoute",
    "deletePublishedServiceRoute",
}


def collect_schema_refs(obj, refs: set):
    """递归收集 $ref 引用的 schema 名称。"""
    if isinstance(obj, dict):
        if "$ref" in obj:
            ref = obj["$ref"]
            if ref.startswith("#/components/schemas/"):
                refs.add(ref.split("/")[-1])
        for v in obj.values():
            collect_schema_refs(v, refs)
    elif isinstance(obj, list):
        for item in obj:
            collect_schema_refs(item, refs)


def resolve_schemas(all_schemas: dict, needed: set) -> dict:
    """递归解析所有被引用的 schema（含嵌套引用）。"""
    resolved = {}
    queue = list(needed)
    while queue:
        name = queue.pop()
        if name in resolved or name not in all_schemas:
            continue
        schema = all_schemas[name]
        resolved[name] = schema
        refs = set()
        collect_schema_refs(schema, refs)
        queue.extend(refs - set(resolved))
    return resolved


def extract(spec_path: str, out_path: str):
    with open(spec_path) as f:
        full = yaml.safe_load(f)

    subset_paths = {}
    needed_refs = set()

    for path, methods in full.get("paths", {}).items():
        subset_methods = {}
        for method, op in methods.items():
            if not isinstance(op, dict):
                continue
            if op.get("operationId") in OPERATION_IDS:
                subset_methods[method] = op
                collect_schema_refs(op, needed_refs)
        if subset_methods:
            # 保留 path-level parameters（如果有）
            if "parameters" in methods:
                subset_methods["parameters"] = methods["parameters"]
                collect_schema_refs(methods["parameters"], needed_refs)
            subset_paths[path] = subset_methods

    if not subset_paths:
        print("ERROR: 未找到任何匹配的 operation，请检查 spec 路径或 OPERATION_IDS", file=sys.stderr)
        sys.exit(1)

    all_schemas = full.get("components", {}).get("schemas", {})
    subset_schemas = resolve_schemas(all_schemas, needed_refs)

    subset = {
        "openapi": full.get("openapi", "3.0.1"),
        "info": {
            "title": "API7 Enterprise - Terraform Provider Subset",
            "version": full.get("info", {}).get("version", ""),
            "description": "Auto-extracted subset for terraform-provider-api7.",
        },
        "servers": full.get("servers", []),
        "security": full.get("security", []),
        "paths": subset_paths,
    }

    components = full.get("components", {})
    if subset_schemas:
        subset["components"] = {
            k: v for k, v in components.items() if k != "schemas"
        }
        subset["components"]["schemas"] = subset_schemas
    elif "securitySchemes" in components:
        subset["components"] = {"securitySchemes": components["securitySchemes"]}

    with open(out_path, "w") as f:
        yaml.dump(subset, f, allow_unicode=True, sort_keys=False, default_flow_style=False)

    print(f"提取完成：{len(subset_paths)} 条路径，{len(subset_schemas)} 个 schema → {out_path}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="提取 Terraform Provider 所需的 OpenAPI 子集")
    parser.add_argument("--spec", required=True, help="完整 OpenAPI 规范文件路径")
    parser.add_argument("--out", default="openapi-subset.yaml", help="输出文件路径")
    args = parser.parse_args()
    extract(args.spec, args.out)
