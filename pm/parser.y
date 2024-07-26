%{
package pm

type pair struct {
  key string
  val interface{}
}

func setResult(l yyLexer, v map[string]interface{}) {
  l.(*lex).result = v
}
%}

%union{
  obj map[string]interface{}
  list []interface{}
  pair pair
  val interface{}
}

%token LexError
%token <val> String Number Literal
%token '{' '}' ',' ':' '[' ']' '.'

%type <obj> object members
%type <pair> pair
%type <val> array
%type <list> elements
%type <val> value

%start start

%%

start: object
  {
    setResult(yylex, $1)
  }

object: '.' '{' members '}'
  {
    $$ = $3
  }
| '{' members '}'
  {
    $$ = $2
  }

members:
  {
    $$ = make(map[string]interface{})
  }
| pair
  {
    $$ = map[string]interface{}{
      $1.key: $1.val,
    }
  }
| members ',' pair
  {
    $1[$3.key] = $3.val
    $$ = $1
  }

pair: String ':' value
  {
    $$ = pair{key: $1.(string), val: $3}
  }

array: '[' elements ']'
  {
    $$ = $2
  }

elements:
  {
    $$ = make([]interface{}, 0)
  }
| value
  {
    $$ = []interface{}{$1}
  }
| elements ',' value
  {
    $$ = append($1, $3)
  }

value:
  String
  {
    $$ = $1
  }
| Number
  {
    $$ = $1
  }
| Literal
  {
    $$ = $1
  }
| object
  {
    $$ = $1
  }
| array
  {
    $$ = $1
  }

%%