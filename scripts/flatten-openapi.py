#!/usr/bin/env python3
"""
Flatten allOf/oneOf schema composition in OpenAPI spec for tfplugingen-openapi.
Only schemas containing allOf/oneOf are flattened; leaf properties are passed through unchanged.
"""

import yaml
import copy
import sys


def needs_flatten(schema):
    return isinstance(schema, dict) and (
        "allOf" in schema or "oneOf" in schema or "anyOf" in schema
    )


def merge_into(target_props, target_required, source):
    """Merge properties and required from source schema into target dicts."""
    if not isinstance(source, dict):
        return
    if "required" in source:
        target_required.update(source["required"])
    if "properties" in source:
        for k, v in source["properties"].items():
            # Recursively flatten if needed, otherwise keep as-is
            target_props[k] = flatten_schema(v) if needs_flatten(v) else v
    for key in ("allOf",):
        for branch in source.get(key, []):
            merge_into(target_props, target_required, branch)
    # For oneOf/anyOf take only the first branch (primary/HTTP case)
    for key in ("oneOf", "anyOf"):
        branches = source.get(key, [])
        if branches:
            merge_into(target_props, target_required, branches[0])


def flatten_schema(schema):
    """Flatten allOf/oneOf/anyOf composition into a single object schema."""
    if not isinstance(schema, dict):
        return schema
    if not needs_flatten(schema):
        return schema

    props = {}
    required = set()

    # Carry over top-level required
    if "required" in schema:
        required.update(schema["required"])

    # Carry over top-level properties
    if "properties" in schema:
        for k, v in schema["properties"].items():
            props[k] = flatten_schema(v) if needs_flatten(v) else v

    for branch in schema.get("allOf", []):
        merge_into(props, required, branch)
    for key in ("oneOf", "anyOf"):
        branches = schema.get(key, [])
        if branches:
            merge_into(props, required, branches[0])

    result = {"type": "object"}
    if schema.get("description"):
        result["description"] = schema["description"]
    if props:
        result["properties"] = props
    if required:
        result["required"] = sorted(required)
    return result


def process_spec(spec):
    spec = copy.deepcopy(spec)
    for path, path_item in spec.get("paths", {}).items():
        for method in ("get", "post", "put", "delete", "patch"):
            op = path_item.get(method)
            if not op:
                continue
            # Flatten request body
            body = op.get("requestBody", {}).get("content", {}).get("application/json", {})
            if "schema" in body and needs_flatten(body["schema"]):
                body["schema"] = flatten_schema(body["schema"])
            # Flatten response bodies
            for status, response in op.get("responses", {}).items():
                rbody = response.get("content", {}).get("application/json", {})
                if "schema" in rbody and needs_flatten(rbody["schema"]):
                    rbody["schema"] = flatten_schema(rbody["schema"])
    return spec


if __name__ == "__main__":
    input_file = sys.argv[1] if len(sys.argv) > 1 else "openapi-subset.yaml"
    output_file = sys.argv[2] if len(sys.argv) > 2 else "openapi-flat.yaml"

    with open(input_file) as f:
        spec = yaml.safe_load(f)

    flattened = process_spec(spec)

    with open(output_file, "w") as f:
        yaml.dump(flattened, f, allow_unicode=True, sort_keys=False)

    print(f"Flattened spec written to {output_file}")
