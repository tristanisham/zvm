%{
package pm

type Node struct {
    Type  string
    Key   string
    Value interface{}
    Items []Node
}

var result Node

func setResult(l yyLexer, v map[string]interface{}) {
  l.(*lex).result = v
}
%}

%union {
    node  Node
    nodes []Node
    str   string
}

%token <str> STRING KEY
%type <node> obj pair value
%type <nodes> pairlist

%%

start
    : obj { result = $1 }
    ;

obj
    : '.' '{' pairlist '}' {
        $$ = Node{Type: "object", Items: $3}
    }
    ;

pairlist
    : pairlist pair {
        $$ = append($1, $2)
    }
    | /* empty */ {
        $$ = []Node{}
    }
    ;

pair
    : '.' KEY '=' value {
        $$ = Node{Type: "pair", Key: $2, Value: $4}
    }
    ;

value
    : STRING {
        $$ = Node{Type: "string", Value: $1}
    }
    | '.' '{' pairlist '}' {
        $$ = Node{Type: "object", Items: $3}
    }
    | /* empty */ {
        $$ = Node{Type: "empty"}
    }
    ;

%%