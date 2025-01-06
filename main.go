package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/aliyun/fc-runtime-go-sdk/fc"
)

type Reader struct {
	pos   int
	slice []string
}

func NewReader(s string) *Reader {
	return &Reader{
		slice: strings.Fields(s),
	}
}

func (r *Reader) GetInt() (int64, error) {
	if r.pos < len(r.slice) {
		val, err := strconv.ParseInt(r.slice[r.pos], 10, 64)
		r.pos++
		return val, err
	}
	return 0, fmt.Errorf("reader out of range: %d", len(r.slice))
}

func (r *Reader) GetUint() (uint64, error) {
	if r.pos < len(r.slice) {
		val, err := strconv.ParseUint(r.slice[r.pos], 10, 64)
		r.pos++
		return val, err
	}
	return 0, fmt.Errorf("reader out of range: %d", len(r.slice))
}

func (r *Reader) GetString() (string, error) {
	if r.pos < len(r.slice) {
		s := r.slice[r.pos]
		r.pos++
		return s, nil
	}
	return "", fmt.Errorf("reader out of range: %d", len(r.slice))
}

type Course struct {
	Code        string
	Term        uint64
	Credits     uint64
	Weight      uint64
	LearnedTime uint64
	Classes     map[string]*Class
	PreCourses  []*Course
	PostCourses []*Course
}

type Class struct {
	Course    *Course
	Code      string
	TimeTable [7]uint64
	WeekTable uint64
}

func ParseInput(r *Reader) (classes map[string]*Class, creditsLimit uint64, termsLimit uint64, err error) {
	courses := make(map[string]*Course)
	classes = make(map[string]*Class)
	coursesCount, err := r.GetUint()
	if err != nil {
		return
	}
	classesCount, err := r.GetUint()
	if err != nil {
		return
	}
	preCoursesCount, err := r.GetUint()
	if err != nil {
		return
	}
	postCoursesCount, err := r.GetUint()
	if err != nil {
		return
	}
	creditsLimit, err = r.GetUint()
	if err != nil {
		return
	}
	termsLimit, err = r.GetUint()
	if err != nil {
		return
	}
	for coursesCount > 0 {
		course := &Course{}
		course.Code, err = r.GetString()
		if err != nil {
			return
		}
		course.Term, err = r.GetUint()
		if err != nil {
			return
		}
		course.Credits, err = r.GetUint()
		if err != nil {
			return
		}
		course.Weight, err = r.GetUint()
		if err != nil {
			return
		}
		if _, exists := courses[course.Code]; exists {
			return
		}
		course.Classes = make(map[string]*Class)
		courses[course.Code] = course
		coursesCount--
	}
	for classesCount > 0 {
		var courseCode string
		class := &Class{}
		courseCode, err = r.GetString()
		if err != nil {
			return
		}
		course, exists := courses[courseCode]
		if !exists || course == nil {
			err = fmt.Errorf("course %s does not exists", courseCode)
			return
		}
		class.Course = course
		class.Code, err = r.GetString()
		if err != nil {
			return
		}
		for i := 0; i < 7; i++ {
			class.TimeTable[i], err = r.GetUint()
			if err != nil {
				return
			}
		}
		class.WeekTable, err = r.GetUint()
		if err != nil {
			return
		}
		code := fmt.Sprintf("%s %s", courseCode, class.Code)
		if _, exists := classes[code]; exists {
			err = fmt.Errorf("class %s already exists", code)
			return
		}
		course.Classes[class.Code] = class
		classes[code] = class
		classesCount--
	}
	for preCoursesCount > 0 {
		var courseCode, preCourseCode string
		courseCode, err = r.GetString()
		if err != nil {
			return
		}
		preCourseCode, err = r.GetString()
		if err != nil {
			return
		}
		course, exists := courses[courseCode]
		if !exists || course == nil {
			err = fmt.Errorf("course %s does not exists", courseCode)
			return
		}
		preCourse, exists := courses[preCourseCode]
		if !exists || preCourse == nil {
			err = fmt.Errorf("pre-course %s does not exists", preCourseCode)
			return
		}
		course.PreCourses = append(course.PreCourses, preCourse)
		preCoursesCount--
	}
	for postCoursesCount > 0 {
		var courseCode, postCourseCode string
		courseCode, err = r.GetString()
		if err != nil {
			return
		}
		postCourseCode, err = r.GetString()
		if err != nil {
			return
		}
		course, exists := courses[courseCode]
		if !exists || course == nil {
			err = fmt.Errorf("course %s does not exists", courseCode)
			return
		}
		postCourse, exists := courses[postCourseCode]
		if !exists || postCourse == nil {
			err = fmt.Errorf("post-course %s does not exists", postCourseCode)
			return
		}
		postCourse.PostCourses = append(course.PostCourses, course)
		postCoursesCount--
	}
	return
}

type ParseOutputResult struct {
	CompulsoryCount  uint64 `json:"compulsory_count"`
	PostCoursesCount uint64 `json:"post_courses_count"`
	OptionalScore    uint64 `json:"optional_score"`
}

func ParseOutput(
	r *Reader,
	classes map[string]*Class,
	creditsLimit uint64,
	termsLimit uint64,
) (
	res ParseOutputResult,
	err error,
) {
	for termsLimit > 0 {
		var classesCount uint64
		classesCount, err = r.GetUint()
		if err != nil {
			return
		}
		currentClasses := []*Class{}
		var termCredits uint64
		for classesCount > 0 {
			var courseCode, classCode string
			courseCode, err = r.GetString()
			if err != nil {
				return
			}
			classCode, err = r.GetString()
			if err != nil {
				return
			}
			code := fmt.Sprintf("%s %s", courseCode, classCode)
			class, exists := classes[code]
			if !exists || class == nil {
				err = fmt.Errorf("class %s does not exists", code)
				return
			}
			course := class.Course
			if course.Term != termsLimit&1 {
				err = fmt.Errorf("the term of course %s is incorrect", courseCode)
				return
			}
			if course.LearnedTime > 0 {
				err = fmt.Errorf("course %s already learned", courseCode)
				return
			}
			if course.Weight > 0 {
				res.OptionalScore += course.Weight
			}
			for _, preCourse := range course.PreCourses {
				if preCourse.LearnedTime == 0 {
					err = fmt.Errorf("pre-course %s not learned", preCourse.Code)
					return
				}
			}
			for _, postCourse := range course.PostCourses {
				if postCourse.LearnedTime == termsLimit+1 {
					res.PostCoursesCount++
				}
			}
			currentClasses = append(currentClasses, class)
			termCredits += course.Credits
			res.CompulsoryCount++
			classesCount--
		}
		if termCredits > creditsLimit {
			err = fmt.Errorf("credits exceeded %v", creditsLimit)
			return
		}
		for i := 0; i < len(currentClasses); i++ {
			for j := i + 1; j < len(currentClasses); j++ {
				classX := currentClasses[i]
				classY := currentClasses[j]
				if classX.WeekTable&classY.WeekTable != 0 {
					for k := 0; k < 7; k++ {
						if classX.TimeTable[k]&classY.TimeTable[k] != 0 {
							err = fmt.Errorf(
								"class %s %s and class %s %s conflicts",
								classX.Course.Code, classX.Code,
								classY.Course.Code, classY.Code,
							)
							return
						}
					}
				}
			}
		}
		for _, class := range currentClasses {
			class.Course.LearnedTime = termsLimit
		}
		termsLimit--
	}
	return
}

type HTTPTriggerEvent struct {
	Version         *string           `json:"version"`
	RawPath         *string           `json:"rawPath"`
	Headers         map[string]string `json:"headers"`
	QueryParameters map[string]string `json:"queryParameters"`
	Body            *string           `json:"body"`
	IsBase64Encoded *bool             `json:"isBase64Encoded"`
	RequestContext  *struct {
		AccountId    string `json:"accountId"`
		DomainName   string `json:"domainName"`
		DomainPrefix string `json:"domainPrefix"`
		RequestId    string `json:"requestId"`
		Time         string `json:"time"`
		TimeEpoch    string `json:"timeEpoch"`
		Http         struct {
			Method    string `json:"method"`
			Path      string `json:"path"`
			Protocol  string `json:"protocol"`
			SourceIp  string `json:"sourceIp"`
			UserAgent string `json:"userAgent"`
		} `json:"http"`
	} `json:"requestContext"`
}

type HTTPTriggerResponse struct {
	StatusCode      int               `json:"statusCode"`
	Headers         map[string]string `json:"headers,omitempty"`
	IsBase64Encoded bool              `json:"isBase64Encoded,omitempty"`
	Body            string            `json:"body"`
}

type RequestBody struct {
	InputData  string `json:"input_data"`
	OutputData string `json:"output_data"`
}

type FailResponseBody struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func InternalServerError() *HTTPTriggerResponse {
	body := &FailResponseBody{
		Success: false,
		Code:    http.StatusInternalServerError,
		Message: "internal server error",
	}
	bytes, _ := json.Marshal(body)
	return &HTTPTriggerResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       string(bytes),
	}
}

func BadRequest(msg string) *HTTPTriggerResponse {
	body := &FailResponseBody{
		Success: false,
		Code:    http.StatusBadRequest,
		Message: msg,
	}
	bytes, err := json.Marshal(body)
	if err != nil {
		return InternalServerError()
	}
	return &HTTPTriggerResponse{
		StatusCode: http.StatusBadRequest,
		Body:       string(bytes),
	}
}

type SuccessResponseBody struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any
}

func Ok(data any) *HTTPTriggerResponse {
	body := &SuccessResponseBody{
		Success: true,
		Code:    http.StatusOK,
		Message: "ok",
		Data:    data,
	}
	bytes, err := json.Marshal(body)
	if err != nil {
		return InternalServerError()
	}
	return &HTTPTriggerResponse{
		StatusCode: http.StatusOK,
		Body:       string(bytes),
	}
}

type JudgeResult struct {
	CompulsoryCount  uint64 `json:"compulsory_count"`
	PostCoursesCount uint64 `json:"post_courses_count"`
	OptionalScore    uint64 `json:"optional_score"`
	Status           uint64 `json:"status"`
	Comment          string `json:"comment"`
}

func HandleRequest(event HTTPTriggerEvent) (*HTTPTriggerResponse, error) {
	reqBody := &RequestBody{}

	if event.Body == nil {
		return BadRequest("the request did not come from an HTTP Trigger"), nil
	}

	if event.IsBase64Encoded != nil && *event.IsBase64Encoded {
		decodedByte, err := base64.StdEncoding.DecodeString(*event.Body)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(decodedByte, reqBody)
		if err != nil {
			return BadRequest(err.Error()), nil
		}
	} else {
		err := json.Unmarshal([]byte(*event.Body), reqBody)
		if err != nil {
			return BadRequest(err.Error()), nil
		}
	}

	inputReader := NewReader(reqBody.InputData)
	outputReader := NewReader(reqBody.OutputData)
	classes, creditsLimit, termsLimit, err := ParseInput(inputReader)
	if err != nil {
		return Ok(&JudgeResult{
			Status:  3,
			Comment: err.Error(),
		}), nil
	}
	res, err := ParseOutput(outputReader, classes, creditsLimit, termsLimit)
	if err != nil {
		fmt.Println("Output Data Error: %v", err)
		return Ok(&JudgeResult{
			Status:  2,
			Comment: err.Error(),
		}), nil
	}
	return Ok(&JudgeResult{
		CompulsoryCount:  res.CompulsoryCount,
		PostCoursesCount: res.PostCoursesCount,
		OptionalScore:    res.OptionalScore,
		Status:           1,
		Comment:          "accepted",
	}), nil
}

func main() {
	fc.Start(HandleRequest)
}
