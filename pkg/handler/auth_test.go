package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/assert/v2"
	"github.com/golang/mock/gomock"
	"github.com/lypolix/todo-app"
	"github.com/lypolix/todo-app/pkg/handler"
	"github.com/lypolix/todo-app/pkg/service"
	mock_service "github.com/lypolix/todo-app/pkg/service/mocks"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/nacl/auth"
	"golang.org/x/tools/go/expect"
)

func TestHandler_signUp(t *testing.T) {
	type mockBehavior func(s *mock_service.MockAuthorization, user todo.User)


	testTable := []struct{
		name string
		inputBody string
		inputUser todo.User
		mockBehavior mockBehavior
		expectedStatusCode int
		expectedRequestBody string
	} {
		{
			name:"OK",
			inputBody: `{"name":"Test","username":"test","password":"qwerty"}`,
			inputUser: todo.User{
				Name: "Test",
				Username: "test",
				Password: "qwerty",
			},
			mockBehavior: func(authorization *mock_service.MockAuthorization, user todo.User){
				authorization.EXPECT(user).Return(1, nil)
			},
			expectedStatusCode: 200,
			expectedRequestBody: `{"id":1}`,
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			auth := mock_service.NewMockAuthorization(c)
			testCase.mockBehavior(auth, testCase.inputUser)

			service := &service.Service{Authorization: auth}
			handler := NewHandler(service)


			r := gin.New()
			r.POST("/sign-up", handler.signUp)


			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/sign-up", bytes.NewBufferString(testCase.inputBody))


			r.ServeHTTP(w, req)

			assert.Equal(t, testCase.expectedStatusCode, w.Code)
			assert.Equal(t, testCase.expectedRequestBody, w.Body.String())
		})
	}
}

