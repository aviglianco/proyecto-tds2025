/**
 * @file Parser grammar for tree-sitter (preprojectlang)
 * @author Agus
 * @license MIT
 */

/// <reference types="tree-sitter-cli/dsl" />
// @ts-check

const commaSepTrailing = (rule) =>
  optional(seq(rule, repeat(seq(",", rule)), optional(",")));

// Operator precedence
const PREC = {
  UNARY: 7, // -x, !x
  MUL: 6, // *, /, %
  ADD: 5, // +, -
  REL: 4, // <, >
  EQ: 3, // ==
  AND: 2, // &&
  OR: 1, // ||
};

export default grammar({
  name: "preprojectlang",

  extras: ($) => [/\s/, $.comment],

  rules: {
    // ────────────────────────────────────────────────────────────────────────────
    // Entry point
    // ────────────────────────────────────────────────────────────────────────────
    source_file: ($) => $.program,

    program: ($) =>
      seq(
        "program",
        "{",
        repeat($.declaration_statement),
        repeat($.method_declaration_statement),
        "}"
      ),

    // ────────────────────────────────────────────────────────────────────────────
    // Types
    // ────────────────────────────────────────────────────────────────────────────
    _void_type: (_$) => "void",
    _bool_type: (_$) => "bool",
    _int_type: (_$) => "integer",
    _type: ($) => choice($._int_type, $._bool_type),

    // ────────────────────────────────────────────────────────────────────────────
    // Blocks & statements
    // ────────────────────────────────────────────────────────────────────────────
    block: ($) =>
      seq(
        "{",
        repeat(field("declaration", $.declaration_statement)),
        repeat(field("statement", $._statement)),
        "}"
      ),

    method_call: ($) =>
      seq($.identifier, "(", commaSepTrailing($._expression), ")"),

    _statement: ($) =>
      choice(
        ";",
        seq($.assignment_statement, ";"),
        seq($.method_call, ";"),
        seq($.return_statement, ";"),
        $.if_statement,
        $.while_statement,
        $.block
      ),

    while_statement: ($) =>
      seq("while", field("condition", $._expression), $.block),

    if_statement: ($) =>
      seq(
        "if",
        "(",
        field("condition", $._expression),
        ")",
        "then",
        $.block,
        optional(seq("else", $.block))
      ),

    declaration_statement: ($) =>
      seq(
        field("type", $._type),
        field("identifier", $.identifier),
        "=",
        field("value", $._expression),
        ";"
      ),

    parameter: ($) =>
      seq(field("type", $._type), field("identifier", $.identifier)),

    method_declaration_statement: ($) =>
      seq(
        field("type", choice($._type, $._void_type)),
        field("identifier", $.identifier),
        "(",
        commaSepTrailing($.parameter),
        ")",
        choice($.block, seq("extern", ";"))
      ),

    assignment_statement: ($) =>
      seq(
        field("identifier", $.identifier),
        "=",
        field("value", $._expression)
      ),

    return_statement: ($) =>
      seq("return", optional(field("value", $._expression))),

    // ────────────────────────────────────────────────────────────────────────────
    // Expressions
    // ────────────────────────────────────────────────────────────────────────────
    paren_expr: ($) => seq("(", $._expression, ")"),

    _expression: ($) => choice($._exp, $.paren_expr),

    _exp: ($) =>
      choice(
        // Unary (highest among operators)
        $.minus,
        $.bool_not,

        // Binary operator groups (with explicit precedence)
        $._int_operation,
        $._rel_operation,
        $._eq_operation,
        $._bool_operation,

        // Primaries
        $.method_call, // before identifier to favor calls when "(" follows
        $.identifier,
        $.num,
        $._bool_const
      ),

    // Equality (named node for the builder)
    _eq_operation: ($) => $.rel_eq,

    rel_eq: ($) =>
      prec.left(
        PREC.EQ,
        seq(field("left", $._expression), "==", field("right", $._expression))
      ),

    // Relational (<, >) (named nodes for the builder)
    _rel_operation: ($) => choice($.rel_lt, $.rel_gt),

    rel_lt: ($) =>
      prec.left(
        PREC.REL,
        seq(field("left", $._expression), "<", field("right", $._expression))
      ),

    rel_gt: ($) =>
      prec.left(
        PREC.REL,
        seq(field("left", $._expression), ">", field("right", $._expression))
      ),

    // Boolean ops (OR lowest)
    _bool_operation: ($) => choice($.bool_conjunction, $.bool_disjunction),

    bool_conjunction: ($) =>
      prec.left(
        PREC.AND,
        seq(field("left", $._expression), "&&", field("right", $._expression))
      ),

    bool_disjunction: ($) =>
      prec.left(
        PREC.OR,
        seq(field("left", $._expression), "||", field("right", $._expression))
      ),

    bool_not: ($) => prec.right(PREC.UNARY, seq("!", $._expression)),

    // Integer ops
    _int_operation: ($) =>
      choice($.int_prod, $.int_div, $.int_rem, $.int_sum, $.int_sub),

    int_prod: ($) =>
      prec.left(
        PREC.MUL,
        seq(field("left", $._expression), "*", field("right", $._expression))
      ),

    int_div: ($) =>
      prec.left(
        PREC.MUL,
        seq(field("left", $._expression), "/", field("right", $._expression))
      ),

    int_rem: ($) =>
      prec.left(
        PREC.MUL,
        seq(field("left", $._expression), "%", field("right", $._expression))
      ),

    int_sum: ($) =>
      prec.left(
        PREC.ADD,
        seq(field("left", $._expression), "+", field("right", $._expression))
      ),

    int_sub: ($) =>
      prec.left(
        PREC.ADD,
        seq(field("left", $._expression), "-", field("right", $._expression))
      ),

    minus: ($) => prec.right(PREC.UNARY, seq("-", $._expression)),

    // ────────────────────────────────────────────────────────────────────────────
    // Terminals
    // ────────────────────────────────────────────────────────────────────────────
    identifier: (_$) => /[a-zA-Z_][a-zA-Z_0-9]*/,

    true: (_$) => "true",
    false: (_$) => "false",
    _bool_const: ($) => choice($.true, $.false),

    num: (_$) => /\d+/,

    comment: ($) =>
      token(
        choice(seq("//", /.*/), seq("/*", /[^*]*\*+([^/*][^*]*\*+)*/, "/"))
      ),
  },
});
