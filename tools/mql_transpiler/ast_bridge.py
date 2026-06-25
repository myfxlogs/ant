"""CST→internal-AST bridge (T2).

Converts tree-sitter CST into ``ast_nodes`` types for codegen consumption.
Uses ``tree_sitter_parser`` as the single .so loader — no duplicated init.

Architecture:
  tree-sitter-c grammar (grammar/mql/grammar.js, 1474 lines)
    → tree_sitter_parser (single .so loader)
    → CSTBridge (this module)
    → ast_nodes (pure data types)
    → ASTTranspiler (ast_transpiler.py codegen)
    → Python SDK output
"""

from __future__ import annotations

from typing import Dict, List, Optional, Tuple

import tree_sitter as ts

from tools.mql_transpiler.ast_nodes import (
    ArrayInitExpr,
    AssignmentExpr,
    BinaryOp,
    CallExpr,
    CompoundStmt,
    ExpressionStmt,
    ForStmt,
    FuncDef,
    Identifier,
    IfStmt,
    NumberLiteral,
    ReturnStmt,
    SourceFile,
    StringLiteral,
    SubscriptExpr,
    SwitchStmt,
    TernaryExpr,
    UnaryOp,
    VarDecl,
    WhileStmt,
)


class CSTBridge:
    """Walk a tree-sitter CST and produce internal AST nodes."""

    def __init__(self, source: bytes):
        self._source = source
        self._known_vars: Dict[str, str] = {}  # name → mql_type

    def source_text(self, node: ts.Node) -> str:
        """Get the source text of a node."""
        return self._source[node.start_byte:node.end_byte].decode("utf-8")

    # ── Top level ─────────────────────────────────────────────────────

    def translate(self, root: ts.Node) -> SourceFile:
        """Convert a tree-sitter translation_unit to internal AST."""
        declarations = []
        for i in range(root.named_child_count):
            child = root.named_child(i)
            decl = self._translate_top_level(child)
            if decl is not None:
                declarations.append(decl)
        return SourceFile(declarations=declarations)

    def _translate_top_level(self, node: ts.Node):
        """Dispatch top-level declaration."""
        t = node.type
        if t == "function_definition":
            return self._translate_function(node)
        if t == "declaration":
            return self._translate_declaration(node)
        if t == "preproc_def":
            return self._translate_preproc_def(node)
        if t == "comment":
            return None  # Top-level comments are documentation
        # For anything else (preproc_include, etc.), skip.
        return None

    # ── Functions ─────────────────────────────────────────────────────

    def _translate_function(self, node: ts.Node) -> FuncDef:
        """Translate a function_definition node."""
        ret_type = "void"
        name = "unknown"
        params: List[str] = []
        body = None

        for i in range(node.named_child_count):
            child = node.named_child(i)
            ct = child.type
            if ct in ("primitive_type", "type_identifier", "sized_type_specifier",
                       "macro_type_specifier"):
                ret_type = self.source_text(child)
            elif ct == "function_declarator":
                # function_declarator has: identifier + parameter_list
                for j in range(child.named_child_count):
                    gc = child.named_child(j)
                    if gc.type in ("identifier", "field_identifier", "statement_identifier"):
                        name = self.source_text(gc)
                    elif gc.type == "parameter_list":
                        params = self._translate_params(gc)
            elif ct == "identifier" or ct == "field_identifier" or ct == "statement_identifier":
                name = self.source_text(child)
            elif ct == "parameter_list":
                params = self._translate_params(child)
            elif ct == "compound_statement":
                body = self._translate_compound(child)
            elif ct in ("pointer_declarator", "array_declarator"):
                # MQL: ignore pointer/array declarators on return type
                pass

        return FuncDef(name=name, return_type=ret_type, params=params, body=body)

    def _translate_params(self, node: ts.Node) -> List[str]:
        """Translate parameter_list → list of param names."""
        names = []
        for i in range(node.named_child_count):
            child = node.named_child(i)
            # Each param is an optional_declaration or parameter_declaration
            if child.type in ("parameter_declaration", "optional_parameter_declaration"):
                # Walk children to find the identifier (param name)
                param_name = self._find_param_name(child)
                if param_name:
                    names.append(param_name)
            elif child.type == "identifier":
                # K&R style params — just the name
                names.append(self.source_text(child))
        return names

    def _find_param_name(self, node: ts.Node) -> Optional[str]:
        """Find the parameter name inside a parameter_declaration node."""
        for i in range(node.named_child_count):
            child = node.named_child(i)
            if child.type in ("identifier", "field_identifier"):
                return self.source_text(child)
            elif child.type == "pointer_declarator":
                inner = self._find_param_name(child)
                if inner:
                    return inner
            elif child.type == "array_declarator":
                inner = self._find_param_name(child)
                if inner:
                    return inner
        # Fall back to any named leaf.
        for child in node.children:
            if child.type in ("identifier", "field_identifier"):
                return self.source_text(child)
        return None

    # ── Declarations ──────────────────────────────────────────────────

    def _translate_declaration(self, node: ts.Node) -> VarDecl:
        """Translate a declaration (variable or extern/input)."""
        var_type = ""
        name = ""
        value = None
        is_extern = False
        is_input = False

        for i in range(node.named_child_count):
            child = node.named_child(i)
            ct = child.type
            text = self.source_text(child)

            if ct == "storage_class_specifier":
                if text == "extern":
                    is_extern = True
                elif text == "input":
                    is_input = True
                elif text == "static":
                    pass  # MQL: static is usually for local scope, skip

            elif ct in ("primitive_type", "type_identifier", "sized_type_specifier",
                         "macro_type_specifier"):
                var_type = text

            elif ct == "init_declarator":
                # Contains: declarator (identifier) + optional value
                name, value = self._translate_init_declarator(child)

            elif ct == "identifier":
                name = text

        self._known_vars[name] = var_type
        return VarDecl(name=name, var_type=var_type, value=value,
                       is_extern=is_extern, is_input=is_input)

    def _translate_init_declarator(self, node: ts.Node) -> Tuple[str, Optional]:
        """Extract (name, value_expr) from an init_declarator."""
        name = ""
        value = None
        for i in range(node.named_child_count):
            child = node.named_child(i)
            ct = child.type
            if name == "" and ct in ("identifier", "field_identifier"):
                name = self.source_text(child)
            elif ct == "pointer_declarator":
                # Recursively find the identifier inside pointer_decl
                inner = self._find_declarator_name(child)
                if inner and name == "":
                    name = inner
            elif ct == "array_declarator":
                inner = self._find_declarator_name(child)
                if inner and name == "":
                    name = inner
            elif ct in ("number_literal", "string_literal", "call_expression",
                         "binary_expression", "unary_expression", "identifier",
                         "parenthesized_expression", "conditional_expression"):
                value = self._translate_expression(child)
        return name, value

    def _find_declarator_name(self, node: ts.Node) -> str:
        """Recursively find the identifier inside any declarator chain."""
        for i in range(node.named_child_count):
            child = node.named_child(i)
            if child.type in ("identifier", "field_identifier"):
                return self.source_text(child)
            if child.type in ("pointer_declarator", "array_declarator",
                               "function_declarator", "parenthesized_declarator"):
                result = self._find_declarator_name(child)
                if result:
                    return result
        return ""

    # ── Statements ─────────────────────────────────────────────────────

    def _translate_statement(self, node: ts.Node):
        """Translate a statement node."""
        t = node.type
        if t == "compound_statement":
            return self._translate_compound(node)
        if t == "if_statement":
            return self._translate_if(node)
        if t == "for_statement":
            return self._translate_for(node)
        if t == "while_statement":
            return self._translate_while(node)
        if t == "return_statement":
            return self._translate_return(node)
        if t == "expression_statement":
            return self._translate_expr_stmt(node)
        if t == "declaration":
            return self._translate_declaration(node)
        if t == "switch_statement":
            return self._translate_switch(node)
        if t == "break_statement":
            return ExpressionStmt(expr=Identifier(name="break"))
        if t == "continue_statement":
            return ExpressionStmt(expr=Identifier(name="continue"))
        # For anything unrecognized, return expression statement with raw text.
        text = self.source_text(node)
        return ExpressionStmt(expr=Identifier(name=f"__raw__{text[:60]}"))

    def _translate_compound(self, node: ts.Node) -> CompoundStmt:
        stmts = []
        for i in range(node.named_child_count):
            child = node.named_child(i)
            stmt = self._translate_statement(child)
            if stmt is not None:
                stmts.append(stmt)
        return CompoundStmt(statements=stmts)

    def _translate_if(self, node: ts.Node) -> IfStmt:
        condition = None
        then_branch = None
        else_branch = None
        for i in range(node.named_child_count):
            child = node.named_child(i)
            ct = child.type
            if ct == "parenthesized_expression":
                # The condition expression is inside parenthesized_expression
                if child.named_child_count > 0:
                    condition = self._translate_expression(child.named_child(0))
            elif ct == "condition_clause":
                # Alternative structure (some C grammars)
                for j in range(child.named_child_count):
                    gc = child.named_child(j)
                    if gc.type == "parenthesized_expression" and gc.named_child_count > 0:
                        condition = self._translate_expression(gc.named_child(0))
            elif ct == "else_clause":
                # else_clause wraps: else + statement
                for j in range(child.named_child_count):
                    gc = child.named_child(j)
                    if gc.type == "if_statement":
                        else_branch = self._translate_if(gc)
                    elif gc.type not in ("else",):
                        else_branch = self._translate_statement(gc)
            elif ct in ("compound_statement", "expression_statement", "if_statement",
                         "for_statement", "while_statement", "return_statement",
                         "switch_statement", "declaration", "break_statement",
                         "continue_statement"):
                if then_branch is None:
                    then_branch = self._translate_statement(child)
                else:
                    else_branch = self._translate_statement(child)

        return IfStmt(condition=condition, then_branch=then_branch, else_branch=else_branch)

    def _translate_for(self, node: ts.Node) -> ForStmt:
        init = None
        condition = None
        update = None
        body = None
        # tree-sitter-c places for-loop parts directly under for_statement:
        #   "for" "(" declaration? expression? ";" expression? ")" statement
        # There's NO for_header wrapper in the C grammar.
        for i in range(node.named_child_count):
            child = node.named_child(i)
            ct = child.type
            if ct == "for_header":
                parts = self._translate_for_header(child)
                init, condition, update = parts
            elif ct == "declaration":
                init = self._translate_declaration(child)
            elif ct in ("binary_expression", "assignment_expression", "call_expression",
                         "update_expression", "unary_expression", "identifier",
                         "number_literal", "parenthesized_expression"):
                if condition is None:
                    condition = self._translate_expression(child)
                else:
                    update = self._translate_expression(child)
            elif ct == "expression_statement":
                # Could be init (with ;) or just wrap an expression.
                if child.named_child_count > 0:
                    expr = self._translate_expression(child.named_child(0))
                    if init is None:
                        init = expr
                    elif condition is None:
                        condition = expr
                    else:
                        update = expr
            elif ct in ("compound_statement", "expression_statement", "break_statement",
                         "continue_statement"):
                body = self._translate_statement(child)

        # If body not found yet, check non-named children (compound_statement is named).
        if body is None:
            for child in node.children:
                if child.type == "compound_statement":
                    body = self._translate_compound(child)
                    break

        return ForStmt(init=init, condition=condition, update=update, body=body)

    def _translate_for_header(self, node: ts.Node) -> Tuple:
        """Extract (init, condition, update) from for_header."""
        parts = [None, None, None]
        idx = 0
        for i in range(node.named_child_count):
            child = node.named_child(i)
            ct = child.type
            if ct == ";":
                idx += 1
                continue
            if idx < 3:
                if ct == "declaration":
                    parts[idx] = self._translate_declaration(child)
                elif ct == "expression_statement":
                    # The expression inside
                    if child.named_child_count > 0:
                        parts[idx] = self._translate_expression(child.named_child(0))
                elif ct in ("binary_expression", "assignment_expression", "call_expression",
                             "update_expression", "identifier", "number_literal",
                             "unary_expression", "parenthesized_expression"):
                    parts[idx] = self._translate_expression(child)
        return tuple(parts)

    def _translate_while(self, node: ts.Node) -> WhileStmt:
        condition = None
        body = None
        for i in range(node.named_child_count):
            child = node.named_child(i)
            ct = child.type
            if ct == "parenthesized_expression":
                if child.named_child_count > 0:
                    condition = self._translate_expression(child.named_child(0))
            elif ct == "condition_clause":
                for j in range(child.named_child_count):
                    gc = child.named_child(j)
                    if gc.type == "parenthesized_expression" and gc.named_child_count > 0:
                        condition = self._translate_expression(gc.named_child(0))
            elif ct in ("compound_statement", "expression_statement"):
                body = self._translate_statement(child)
        return WhileStmt(condition=condition, body=body)

    def _translate_return(self, node: ts.Node) -> ReturnStmt:
        value = None
        for i in range(node.named_child_count):
            child = node.named_child(i)
            if child.type != ";":
                value = self._translate_expression(child)
        return ReturnStmt(value=value)

    def _translate_expr_stmt(self, node: ts.Node) -> ExpressionStmt:
        expr = None
        for i in range(node.named_child_count):
            child = node.named_child(i)
            if child.type != ";":
                expr = self._translate_expression(child)
        return ExpressionStmt(expr=expr)

    def _translate_switch(self, node: ts.Node) -> SwitchStmt:
        expr = None
        cases = []
        default = None
        for i in range(node.named_child_count):
            child = node.named_child(i)
            ct = child.type
            if ct == "parenthesized_expression" and child.named_child_count > 0:
                expr = self._translate_expression(child.named_child(0))
            elif ct == "switch_body":
                # Process case statements
                current_case_val = None
                current_case_stmts = []
                for j in range(child.named_child_count):
                    gc = child.named_child(j)
                    if gc.type == "case_statement":
                        # Save previous case
                        if current_case_val is not None:
                            cases.append((current_case_val, current_case_stmts))
                        elif current_case_stmts:
                            default = current_case_stmts
                        # Extract case value
                        case_val = None
                        for k in range(gc.named_child_count):
                            kc = gc.named_child(k)
                            if kc.type != "case" and kc.type != ":":
                                case_val = self._translate_expression(kc)
                                break
                        current_case_val = case_val
                        current_case_stmts = []
                        # Add body statements
                        for k in range(gc.named_child_count):
                            kc = gc.named_child(k)
                            if kc.type != "case" and kc.type != ":" and kc.type not in (
                                    "number_literal", "identifier", "string_literal",
                                    "call_expression", "binary_expression", "unary_expression"):
                                stmt = self._translate_statement(kc)
                                if stmt:
                                    current_case_stmts.append(stmt)
                    elif gc.type == "default_statement":
                        if current_case_val is not None:
                            cases.append((current_case_val, current_case_stmts))
                            current_case_val = None
                            current_case_stmts = []
                        default = []
                        for k in range(gc.named_child_count):
                            kc = gc.named_child(k)
                            if kc.type not in ("default", ":"):
                                stmt = self._translate_statement(kc)
                                if stmt:
                                    default.append(stmt)
                    else:
                        stmt = self._translate_statement(gc)
                        if stmt:
                            current_case_stmts.append(stmt)
                # Flush last case
                if current_case_val is not None:
                    cases.append((current_case_val, current_case_stmts))
                elif current_case_stmts:
                    default = current_case_stmts

        return SwitchStmt(expr=expr, cases=cases, default=default)

    # ── Expressions ────────────────────────────────────────────────────

    def _translate_expression(self, node: ts.Node):
        """Translate any expression node."""
        t = node.type
        if t == "identifier" or t == "field_identifier" or t == "statement_identifier" or t == "type_identifier":
            name = self.source_text(node)
            # Map MQL boolean/null literals that tree-sitter tokenizes as identifiers.
            if name in ("true", "True", "TRUE"):
                return Identifier(name="true")
            if name in ("false", "False", "FALSE"):
                return Identifier(name="false")
            if name in ("NULL", "null", "Null"):
                return Identifier(name="NULL")
            return Identifier(name=name)
        if t == "true":
            return Identifier(name="true")
        if t == "false":
            return Identifier(name="false")
        if t == "null" or t == "nullptr":
            return Identifier(name="NULL")
        if t == "number_literal":
            return NumberLiteral(value=self.source_text(node))
        if t in ("string_literal", "system_lib_string"):
            text = self.source_text(node)
            # Strip surrounding quotes
            if len(text) >= 2:
                inner = text[1:-1]
            else:
                inner = text
            return StringLiteral(value=inner)
        if t == "binary_expression":
            return self._translate_binary(node)
        if t == "unary_expression":
            return self._translate_unary(node)
        if t == "call_expression":
            return self._translate_call(node)
        if t == "subscript_expression":
            return self._translate_subscript(node)
        if t == "assignment_expression":
            return self._translate_assignment(node)
        if t == "conditional_expression":
            return self._translate_ternary(node)
        if t == "parenthesized_expression":
            if node.named_child_count > 0:
                return self._translate_expression(node.named_child(0))
        if t == "update_expression":
            # i++ or ++i — treat as binary op
            return self._translate_update(node)
        if t == "field_expression":
            return self._translate_field_expr(node)
        if t == "comma_expression":
            # a, b → just return the last value
            if node.named_child_count > 0:
                return self._translate_expression(node.named_child(node.named_child_count - 1))
        if t == "cast_expression":
            # (type)expr → just return the inner expression (MQL doesn't cast like this)
            for i in range(node.named_child_count):
                child = node.named_child(i)
                if child.type not in ("primitive_type", "type_identifier", "sized_type_specifier"):
                    return self._translate_expression(child)
        if t == "init_declarator":
            name, value = self._translate_init_declarator(node)
            if name:
                return AssignmentExpr(lhs=name, rhs=value)
        if t == "ERROR":
            text = self.source_text(node)[:80]
            return Identifier(name=f"__error__{text}")
        # Fallback: return as raw identifier.
        return Identifier(name=f"__raw__{self.source_text(node)[:60]}")

    def _translate_binary(self, node: ts.Node) -> BinaryOp:
        left = None
        op = ""
        right = None
        for i in range(node.named_child_count):
            child = node.named_child(i)
            ct = child.type
            if ct in ("+", "-", "*", "/", "%", "==", "!=", "<", ">", "<=", ">=",
                       "&&", "||", "&", "|", "^", "<<", ">>"):
                op = ct
            elif left is None:
                left = self._translate_expression(child)
            else:
                right = self._translate_expression(child)
        # If the operator is an anonymous child
        if not op:
            for child in node.children:
                if child.type in ("+", "-", "*", "/", "%", "==", "!=", "<", ">",
                                   "<=", ">=", "&&", "||", "&", "|", "^", "<<", ">>"):
                    op = child.type
                    break
        return BinaryOp(left=left, op=op, right=right)

    def _translate_unary(self, node: ts.Node) -> UnaryOp:
        op = ""
        operand = None
        for child in node.children:
            if child.type in ("-", "+", "!", "~", "&", "*"):
                op = child.type
                if child.is_named:
                    operand = self._translate_expression(child)
        for i in range(node.named_child_count):
            child = node.named_child(i)
            if child.type not in ("-", "+", "!", "~", "&", "*"):
                operand = self._translate_expression(child)
        return UnaryOp(op=op, operand=operand)

    def _translate_call(self, node: ts.Node) -> CallExpr:
        name = ""
        args = []
        for i in range(node.named_child_count):
            child = node.named_child(i)
            ct = child.type
            if ct in ("identifier", "field_identifier"):
                name = self.source_text(child)
            elif ct == "argument_list":
                # Extract comma-separated arguments
                for j in range(child.named_child_count):
                    gc = child.named_child(j)
                    if gc.type != ",":
                        args.append(self._translate_expression(gc))
            elif ct == "field_expression":
                # e.g. obj.method() → name = "obj.method"
                name = self.source_text(child)
        return CallExpr(name=name, args=args)

    def _translate_subscript(self, node: ts.Node) -> SubscriptExpr:
        name = ""
        index = None
        for i in range(node.named_child_count):
            child = node.named_child(i)
            ct = child.type
            if ct in ("identifier", "field_identifier"):
                name = self.source_text(child)
            elif ct != "[" and ct != "]":
                index = self._translate_expression(child)
        return SubscriptExpr(name=name, index=index)

    def _translate_assignment(self, node: ts.Node) -> AssignmentExpr:
        lhs = ""
        rhs = None
        op = "="
        for i in range(node.named_child_count):
            child = node.named_child(i)
            ct = child.type
            if lhs == "" and ct in ("identifier", "field_identifier",
                                     "subscript_expression", "field_expression"):
                lhs = self.source_text(child)
            elif ct in ("=", "+=", "-=", "*=", "/=", "%="):
                op = ct
            else:
                # Everything else (after lhs is set) is rhs
                rhs = self._translate_expression(child)
        return AssignmentExpr(lhs=lhs, rhs=rhs)

    def _translate_ternary(self, node: ts.Node) -> TernaryExpr:
        """conditional_expression: condition ? true_val : false_val."""
        condition = None
        true_val = None
        false_val = None
        stage = 0  # 0=cond, 1=true, 2=false
        for i in range(node.named_child_count):
            child = node.named_child(i)
            if child.type == "?":
                stage = 1
            elif child.type == ":":
                stage = 2
            elif stage == 0:
                condition = self._translate_expression(child)
            elif stage == 1:
                true_val = self._translate_expression(child)
            else:
                false_val = self._translate_expression(child)
        return TernaryExpr(condition=condition, true_val=true_val, false_val=false_val)

    def _translate_update(self, node: ts.Node):
        """i++ or ++i → BinaryOp(i, '+', 1)."""
        operand = None
        op = "+"
        for child in node.children:
            if child.type in ("++", "--"):
                op = child.type[0]  # '+' or '-'
            elif child.is_named:
                operand = self._translate_expression(child)
        if operand:
            return BinaryOp(left=operand, op=op, right=NumberLiteral(value="1"))
        return Identifier(name="__update__")

    def _translate_field_expr(self, node: ts.Node):
        """obj.member → Identifier('obj.member')."""
        parts = []
        for child in node.children:
            if child.type in ("identifier", "field_identifier"):
                parts.append(self.source_text(child))
            elif child.type == "call_expression":
                parts.append(self.source_text(child))
        return Identifier(name=".".join(parts) if parts else self.source_text(node))

    # ── Preprocessor ───────────────────────────────────────────────────

    def _translate_preproc_def(self, node: ts.Node):
        """#define NAME value"""
        name = ""
        value_text = ""
        for i in range(node.named_child_count):
            child = node.named_child(i)
            if child.type == "identifier":
                name = self.source_text(child)
            elif child.type == "preproc_arg":
                value_text = self.source_text(child)
        if name:
            self._known_vars[name] = value_text or "1"
        return None  # #define is metadata, not a declaration

    # ── Metadata access ────────────────────────────────────────────────

    @property
    def global_vars(self) -> List[Tuple[str, str]]:
        """Return known #define macros as (name, value) pairs."""
        return [(k, v or "1") for k, v in self._known_vars.items()]


def parse_mql(source: str) -> SourceFile:
    """Parse MQL source and return internal AST.

    This is the SINGLE parser entry point (ADR-0020 D8).  Uses
    ``tree_sitter_parser`` for grammar loading — no duplicated .so init.
    """
    from tools.mql_transpiler.tree_sitter_parser import available, parse as ts_parse

    if not available():
        raise RuntimeError(
            "tree-sitter MQL grammar not available. "
            "Build it with: cd grammar/mql && npx tree-sitter generate "
            "&& gcc -shared -fPIC -o mql.so src/parser.c -I src"
        )

    tree = ts_parse(source)
    if tree is None:
        raise RuntimeError("tree-sitter parse returned None")

    bridge = CSTBridge(source.encode("utf-8"))
    return bridge.translate(tree.root_node)


# Backward-compat alias.
parse_mql_tree_sitter = parse_mql
