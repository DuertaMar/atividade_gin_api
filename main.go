package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Database struct {
	Alunos []Aluno
	Salas  []Sala
	Turmas []Turma
}

func main() {

	r := gin.New()

	db := &Database{
		Alunos: []Aluno{},
		Salas:  []Sala{},
		Turmas: []Turma{},
	}
	alunoController := NovoAlunoController(db)
	salaController := NovoSalaController(db)
	turmaController := NovoTurmaController(db)

	// Uso dos Middlewares globais nativos e personalizados
	r.Use(gin.Recovery())

	// 4. Mapeamento de Rotas sob Grupo Versionado
	v1 := r.Group("/api/v1")
	{
		// Monitoramento da API
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":    "healthy",
				"timestamp": time.Now(),
				"version":   "1.0.0",
			})
		})
		// Domínio de Alunos
		alunosGroup := v1.Group("/alunos")
		{
			alunosGroup.POST("", alunoController.Criar)
			alunosGroup.GET("", alunoController.Listar)
			alunosGroup.GET("/:matricula", alunoController.BuscarPorMatricula)
			alunosGroup.PUT("/:matricula", alunoController.Atualizar)
			alunosGroup.DELETE("/:matricula", alunoController.Deletar)
		}

		// Domínio de Salas
		salasGroup := v1.Group("/salas")
		{
			salasGroup.POST("", salaController.Criar)
			salasGroup.GET("", salaController.Listar)
			salasGroup.GET("/:id/grade", salaController.ConsultarGrade)
			salasGroup.PUT("/:id", salaController.Atualizar)
			salasGroup.PATCH("/:id/inativar", salaController.Inativar)
			salasGroup.PATCH("/:id/ativar", salaController.Ativar)
		}

		// Domínio de Turmas (Classes)
		turmasGroup := v1.Group("/turmas")
		{
			//Turma
			turmasGroup.POST("", turmaController.Criar)
			turmasGroup.GET("", turmaController.Listar)
			turmasGroup.PATCH("/:id/inativar", turmaController.Inativar)
			//Aluno
			turmasGroup.GET("/:id/alunos", turmaController.ListarAlunos)
			turmasGroup.POST("/:id/alunos", turmaController.MatricularAluno)
			turmasGroup.DELETE("/:id/alunos/:matricula", turmaController.RemoverAluno)
			//Alocação de sala
			turmasGroup.POST("/:id/alocacoes", turmaController.AlocarSala)
			turmasGroup.DELETE("/:id/alocacoes/:indexAlocacao", turmaController.RemoverAlocacao)
		}
	}

	r.Run(":8080")
}

func ValidarHorario(horario string) bool {
	if len(horario) != 5 || !strings.Contains(horario, ":") {
		return false
	}

	partes := strings.Split(horario, ":")
	if len(partes) != 2 {
		return false
	}

	hora, errH := strconv.Atoi(partes[0])
	minuto, errM := strconv.Atoi(partes[1])

	if errH != nil || errM != nil {
		return false
	}

	if hora < 0 || hora > 23 || minuto < 0 || minuto > 59 {
		return false
	}

	return true
}
