/**
 * @file Parser grammar for tree-sitter
 * @author Agus
 * @license MIT
 */

/// <reference types="tree-sitter-cli/dsl" />
// @ts-check

const commaSeparatedOptional = (rule) =>
  optional(seq(repeat(seq(rule, ",")), rule));

export default grammar({
  name: "preprojectlang",

  extras: ($) => [/\s/, $.comment],

  rules: {
    // ────────────────────────────────────────────────────────────────────────────
    // Entry points
    // ────────────────────────────────────────────────────────────────────────────
    //source_file: ($) =>
    //  choice(seq(choice($._void_type, $._int_type, $._bool_type), $.main)),

    source_file: ($) => $.program,

    program: ($) =>
      seq(
        "program",
        "{",
        repeat($.declaration_statement),
        repeat($.method_declaration_statement),
        "}"
      ),

    main: ($) => seq("main", field("args", $.args), field("block", $.block)),

    // ────────────────────────────────────────────────────────────────────────────
    // Function arguments
    // ────────────────────────────────────────────────────────────────────────────
    arg: (_$) => "arg",
    args: ($) => seq("(", optional(seq(repeat(seq($.arg, ",")), $.arg)), ")"),

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
      seq($.identifier, "(", commaSeparatedOptional($._expression), ")"),

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

    while_statement: ($) => seq("while", "(", $._expression, ")", $.block),

    if_statement: ($) =>
      seq(
        "if",
        "(",
        $._expression,
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
        seq("(", commaSeparatedOptional($.parameter), ")"),
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
    _expression: ($) => choice($._exp, seq("(", $._expression, ")")),

    _exp: ($) =>
      prec.left(
        choice(
          $._int_operation,
          $._rel_operation,
          $._bool_operation,
          $.num,
          $._bool_const,
          $.identifier,
          $.method_call,
          $.minus,
          $.bool_not
          
        )
      ),

    _rel_operation: ($) => choice($.rel_gt, $.rel_lt, $.rel_eq, $.int_rem),

    rel_eq: ($) =>
      prec.left(
        seq(field("left", $._expression), "==", field("right", $._expression))
      ),

    rel_lt: ($) =>
      prec.left(
        seq(field("left", $._expression), "<", field("right", $._expression))
      ),

    rel_gt: ($) =>
      prec.left(
        seq(field("left", $._expression), ">", field("right", $._expression))
      ),

    _bool_operation: ($) => choice($.bool_conjunction, $.bool_disjunction),

    bool_conjunction: ($) =>
      prec.left(
        seq(field("left", $._expression), "&&", field("right", $._expression))
      ),

    bool_disjunction: ($) =>
      prec.left(
        seq(field("left", $._expression), "||", field("right", $._expression))
      ),
    
    bool_not: ($) => prec.right(2, seq("!", $._expression)),

    _int_operation: ($) => choice($.int_prod, $.int_div, $.int_sum, $.int_sub),

    int_prod: ($) =>
      prec.left(
        1,
        seq(field("left", $._expression), "*", field("right", $._expression))
      ),
    int_div: ($) =>
      prec.left(
        1,
        seq(field("left", $._expression), "/", field("right", $._expression))
      ),
    int_sum: ($) =>
      prec.left(
        seq(field("left", $._expression), "+", field("right", $._expression))
      ),
    int_sub: ($) =>
      prec.left(
        seq(field("left", $._expression), "-", field("right", $._expression))
      ),    
    int_rem: ($) =>
      prec.left(
        seq(field("left", $._expression), "%", field("right", $._expression))
      ),

    minus: ($) => prec.right(2, seq("-", $._expression)),


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
