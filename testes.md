TESTES_API.md
Testes da API

URL utilizada:

http://localhost:8080/api/v1
Health
GET /health

Retorno esperado: 200

Alunos

Criar aluno:

POST /alunos
Content-Type: application/json

{
    "nome": "Gabriel",
    "matricula": 123,
    "usuario_email": "gabriel"
}

Retorno esperado: 201

Listar alunos:

GET /alunos

Retorno esperado: 200

Buscar aluno:

GET /alunos/123

Retorno esperado: 200

Salas

Criar sala:

POST /salas
Content-Type: application/json

{
    "nome": "Sala 1031",
    "capacidade": 30,
    "recursos": ["Computador", "Ar-Condicionado"]
}

Retorno esperado: 201

Listar salas:

GET /salas

Retorno esperado: 200

Turmas

Criar turma:

POST /turmas
Content-Type: application/json

{
    "nome": "Turma A",
    "disciplina": "Desenvolvimento",
    "professor": "Professor 01"
}

Retorno esperado: 201

Listar turmas:

GET /turmas

Retorno esperado: 200

Matrícula

Matricular aluno:

POST /turmas/1/alunos
Content-Type: application/json

{
    "matricula": 123
}

Retorno esperado: 200

Listar alunos da turma:

GET /turmas/1/alunos

Retorno esperado: 200

Alocação

Alocar sala:

POST /turmas/1/alocacoes
Content-Type: application/json

{
    "sala_id": 1,
    "dia_da_semana": "Segunda-feira",
    "horario_inicio": "19:00",
    "horario_fim": "20:30"
}

Retorno esperado: 201

Consultar grade:

GET /salas/1/grade

Retorno esperado: 200

Alguns erros testados
Aluno com matrícula repetida → 409
Aluno inexistente → 404
Sala com capacidade insuficiente → 422
Sala ocupada no mesmo horário → 409
Aluno com conflito de horário → 409
Dados inválidos → 400