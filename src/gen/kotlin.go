package gen

import (
	"fmt"
	"strings"

	"github.com/benn-herrera/xplatter/model"
	"github.com/benn-herrera/xplatter/resolver"
)

func init() {
	Register("kotlin", func() Generator { return &KotlinGenerator{} })
}

// KotlinGenerator produces a Kotlin public API file and a JNI C bridge file.
type KotlinGenerator struct{}

func (g *KotlinGenerator) Name() string { return "kotlin" }

func (g *KotlinGenerator) Generate(ctx *Context) ([]*OutputFile, error) {
	api := ctx.API
	apiName := api.API.Name
	pascalName := ToPascalCase(apiName)
	packageName := strings.ReplaceAll(apiName, "_", ".")

	ktHeader := GeneratedFileHeader(ctx, "//", false)
	jniHeader := GeneratedFileHeaderBlock(ctx, false)

	ktContent, err := generateKotlinFile(api, ctx.ResolvedTypes, pascalName, packageName)
	if err != nil {
		return nil, fmt.Errorf("generating Kotlin file: %w", err)
	}

	jniContent, err := generateJNIFile(api, ctx.ResolvedTypes, pascalName, packageName)
	if err != nil {
		return nil, fmt.Errorf("generating JNI C bridge: %w", err)
	}

	return []*OutputFile{
		{Path: pascalName + ".kt", Content: []byte(ktHeader + "\n" + ktContent)},
		{Path: apiName + "_jni.c", Content: []byte(jniHeader + "\n" + jniContent)},
	}, nil
}

// ---------- Kotlin file generation ----------

func generateKotlinFile(api *model.APIDefinition, resolved resolver.ResolvedTypes, pascalName, packageName string) (string, error) {
	var b strings.Builder

	// Package and imports
	fmt.Fprintf(&b, "package %s\n\n", packageName)

	// Error exception class — collect all unique error types
	errorTypes := CollectErrorTypes(api)
	for _, errType := range errorTypes {
		writeKotlinException(&b, errType)
	}

	// Data classes for FlatBuffer return types
	writeKotlinFBSDataClasses(&b, resolved, api)

	// Handle wrapper classes
	for _, h := range api.Handles {
		writeKotlinHandleClass(&b, h, api, pascalName)
	}

	// Singleton object for the native library and methods without handles
	writeKotlinNativeObject(&b, api, pascalName)

	return b.String(), nil
}

// writeKotlinException writes a Kotlin exception class for a FlatBuffer error enum.
func writeKotlinException(b *strings.Builder, errType string) {
	className := kotlinErrorExceptionName(errType)
	fmt.Fprintf(b, "class %s(val errorCode: Int) : Exception(\"Error code: $errorCode\")\n\n", className)
}

// kotlinErrorExceptionName converts a FlatBuffer error type (e.g., "Common.ErrorCode")
// to a Kotlin exception class name (e.g., "CommonErrorCodeException").
func kotlinErrorExceptionName(errType string) string {
	return strings.ReplaceAll(errType, ".", "") + "Exception"
}

// writeKotlinHandleClass writes a Kotlin wrapper class for an opaque handle.
func writeKotlinHandleClass(b *strings.Builder, h model.HandleDef, api *model.APIDefinition, pascalName string) {
	className := h.Name

	if h.Description != "" {
		fmt.Fprintf(b, "/**\n * %s\n */\n", h.Description)
	}
	fmt.Fprintf(b, "class %s internal constructor(internal val handle: Long) : AutoCloseable {\n", className)

	// Find methods that take this handle as the first parameter (instance methods)
	for _, iface := range api.Interfaces {
		for _, method := range iface.Methods {
			if isInstanceMethod(method, h.Name) {
				writeKotlinInstanceMethod(b, iface.Name, &method, pascalName)
			}
		}
	}

	// AutoCloseable close method — find the interface that constructs this handle
	destructorIfaceName := ""
	for i := range api.Interfaces {
		if ifaceHandleName, ok := api.Interfaces[i].ConstructorHandleName(); ok && ifaceHandleName == h.Name {
			destructorIfaceName = api.Interfaces[i].Name
			break
		}
	}
	if destructorIfaceName != "" {
		destroyMethodName := DestructorMethodName(h.Name)
		fmt.Fprintf(b, "    override fun close() {\n")
		fmt.Fprintf(b, "        %s.%s(handle)\n", pascalName, jniNativeMethodName(destructorIfaceName, destroyMethodName))
		fmt.Fprintf(b, "    }\n")
	} else {
		fmt.Fprintf(b, "    override fun close() { }\n")
	}

	fmt.Fprintf(b, "}\n\n")
}

// isInstanceMethod returns true if the method's first parameter is a handle of the given type.
func isInstanceMethod(method model.MethodDef, handleName string) bool {
	if len(method.Parameters) == 0 {
		return false
	}
	first := method.Parameters[0]
	hName, ok := model.IsHandle(first.Type)
	return ok && hName == handleName
}

// writeKotlinInstanceMethod writes a Kotlin method on a handle wrapper class.
func writeKotlinInstanceMethod(b *strings.Builder, ifaceName string, method *model.MethodDef, pascalName string) {
	methodName := ToCamelCase(method.Name)
	nativeName := jniNativeMethodName(ifaceName, method.Name)
	hasError := method.Error != ""
	hasReturn := method.Returns != nil

	// Build Kotlin parameters (skip the first handle param — it's 'this')
	var ktParams []string
	var nativeCallArgs []string
	nativeCallArgs = append(nativeCallArgs, "handle")

	for _, p := range method.Parameters[1:] {
		ktType := kotlinParamType(p.Type)
		ktParams = append(ktParams, ToCamelCase(p.Name)+": "+ktType)
		nativeCallArgs = append(nativeCallArgs, kotlinParamToNativeArg(p))
	}

	// Determine Kotlin return type
	ktReturnType := ""
	if hasReturn {
		ktReturnType = kotlinReturnType(method.Returns.Type)
	}

	// Method signature
	paramStr := strings.Join(ktParams, ", ")
	if ktReturnType != "" {
		fmt.Fprintf(b, "    fun %s(%s): %s {\n", methodName, paramStr, ktReturnType)
	} else {
		fmt.Fprintf(b, "    fun %s(%s) {\n", methodName, paramStr)
	}

	// Method body
	callArgs := strings.Join(nativeCallArgs, ", ")
	if hasError {
		if hasReturn {
			retType := method.Returns.Type
			if model.IsFlatBufferType(retType) {
				// FlatBuffer return: JNI throws exception, native returns data class directly
				fmt.Fprintf(b, "        return %s.%s(%s)\n", pascalName, nativeName, callArgs)
			} else {
				// Handle or primitive: LongArray pattern [errorCode, result]
				fmt.Fprintf(b, "        val result = %s.%s(%s)\n", pascalName, nativeName, callArgs)
				fmt.Fprintf(b, "        if (result[0] != 0L) throw %s(result[0].toInt())\n",
					kotlinErrorExceptionName(method.Error))
				if _, ok := model.IsHandle(retType); ok {
					fmt.Fprintf(b, "        return %s(result[1])\n", kotlinHandleReturnType(retType))
				} else {
					fmt.Fprintf(b, "        return result[1]\n")
				}
			}
		} else {
			// Fallible without return
			fmt.Fprintf(b, "        val rc = %s.%s(%s)\n", pascalName, nativeName, callArgs)
			fmt.Fprintf(b, "        if (rc != 0) throw %s(rc)\n",
				kotlinErrorExceptionName(method.Error))
		}
	} else {
		if hasReturn {
			retType := method.Returns.Type
			if _, ok := model.IsHandle(retType); ok {
				fmt.Fprintf(b, "        return %s(%s.%s(%s))\n",
					kotlinHandleReturnType(retType), pascalName, nativeName, callArgs)
			} else {
				fmt.Fprintf(b, "        return %s.%s(%s)\n", pascalName, nativeName, callArgs)
			}
		} else {
			fmt.Fprintf(b, "        %s.%s(%s)\n", pascalName, nativeName, callArgs)
		}
	}

	fmt.Fprintf(b, "    }\n\n")
}

// writeKotlinNativeObject writes the companion/singleton object containing native methods
// and factory functions (constructors and non-instance methods).
func writeKotlinNativeObject(b *strings.Builder, api *model.APIDefinition, pascalName string) {
	fmt.Fprintf(b, "object %s {\n", pascalName)
	fmt.Fprintf(b, "    init {\n")
	fmt.Fprintf(b, "        System.loadLibrary(\"%s\")\n", api.API.Name)
	fmt.Fprintf(b, "    }\n\n")

	// Factory methods: explicit constructors from each interface
	for _, iface := range api.Interfaces {
		for i := range iface.Constructors {
			writeKotlinFactoryMethod(b, iface.Name, &iface.Constructors[i], pascalName)
		}
	}

	// Non-lifecycle, non-instance methods (namespace-style static methods)
	for _, iface := range api.Interfaces {
		for i := range iface.Methods {
			if !isAnyInstanceMethod(iface.Methods[i], api) {
				writeKotlinFactoryMethod(b, iface.Name, &iface.Methods[i], pascalName)
			}
		}
	}

	// JNI native method declarations: constructors, auto-destructor, then regular methods
	for _, iface := range api.Interfaces {
		for i := range iface.Constructors {
			writeKotlinNativeDecl(b, iface.Name, &iface.Constructors[i])
		}
		if handleName, ok := iface.ConstructorHandleName(); ok {
			destructor := SyntheticDestructor(handleName)
			writeKotlinNativeDecl(b, iface.Name, &destructor)
		}
		for i := range iface.Methods {
			writeKotlinNativeDecl(b, iface.Name, &iface.Methods[i])
		}
	}

	fmt.Fprintf(b, "}\n")
}

// isAnyInstanceMethod returns true if this method is an instance method on any handle.
func isAnyInstanceMethod(method model.MethodDef, api *model.APIDefinition) bool {
	if len(method.Parameters) == 0 {
		return false
	}
	first := method.Parameters[0]
	hName, ok := model.IsHandle(first.Type)
	if !ok {
		return false
	}
	// Check that this handle is defined in the API
	return api.HandleByName(hName) != nil
}

// writeKotlinFactoryMethod writes a top-level factory method (e.g., createEngine).
func writeKotlinFactoryMethod(b *strings.Builder, ifaceName string, method *model.MethodDef, pascalName string) {
	methodName := ToCamelCase(method.Name)
	nativeName := jniNativeMethodName(ifaceName, method.Name)
	hasError := method.Error != ""
	hasReturn := method.Returns != nil

	var ktParams []string
	var nativeCallArgs []string
	for _, p := range method.Parameters {
		ktType := kotlinParamType(p.Type)
		ktParams = append(ktParams, ToCamelCase(p.Name)+": "+ktType)
		nativeCallArgs = append(nativeCallArgs, kotlinParamToNativeArg(p))
	}

	ktReturnType := ""
	if hasReturn {
		ktReturnType = kotlinReturnType(method.Returns.Type)
	}

	paramStr := strings.Join(ktParams, ", ")
	if ktReturnType != "" {
		fmt.Fprintf(b, "    fun %s(%s): %s {\n", methodName, paramStr, ktReturnType)
	} else {
		fmt.Fprintf(b, "    fun %s(%s) {\n", methodName, paramStr)
	}

	callArgs := strings.Join(nativeCallArgs, ", ")
	if hasError {
		if hasReturn {
			retType := method.Returns.Type
			if model.IsFlatBufferType(retType) {
				// FlatBuffer return: JNI throws exception, native returns data class directly
				fmt.Fprintf(b, "        return %s(%s)\n", nativeName, callArgs)
			} else {
				// Handle or primitive: LongArray pattern
				fmt.Fprintf(b, "        val result = %s(%s)\n", nativeName, callArgs)
				fmt.Fprintf(b, "        if (result[0] != 0L) throw %s(result[0].toInt())\n",
					kotlinErrorExceptionName(method.Error))
				if _, ok := model.IsHandle(retType); ok {
					fmt.Fprintf(b, "        return %s(result[1])\n", kotlinHandleReturnType(retType))
				} else {
					fmt.Fprintf(b, "        return result[1]\n")
				}
			}
		} else {
			fmt.Fprintf(b, "        val rc = %s(%s)\n", nativeName, callArgs)
			fmt.Fprintf(b, "        if (rc != 0) throw %s(rc)\n",
				kotlinErrorExceptionName(method.Error))
		}
	} else {
		if hasReturn {
			retType := method.Returns.Type
			if _, ok := model.IsHandle(retType); ok {
				fmt.Fprintf(b, "        return %s(%s(%s))\n",
					kotlinHandleReturnType(retType), nativeName, callArgs)
			} else {
				fmt.Fprintf(b, "        return %s(%s)\n", nativeName, callArgs)
			}
		} else {
			fmt.Fprintf(b, "        %s(%s)\n", nativeName, callArgs)
		}
	}

	fmt.Fprintf(b, "    }\n\n")
}

// writeKotlinNativeDecl writes a JNI external native method declaration.
func writeKotlinNativeDecl(b *strings.Builder, ifaceName string, method *model.MethodDef) {
	nativeName := jniNativeMethodName(ifaceName, method.Name)
	hasError := method.Error != ""
	hasReturn := method.Returns != nil

	var params []string
	for _, p := range method.Parameters {
		params = append(params, kotlinNativeDeclParam(p))
	}

	var returnType string
	switch {
	case hasError && hasReturn:
		if model.IsFlatBufferType(method.Returns.Type) {
			returnType = kotlinFBSDataClassName(method.Returns.Type)
		} else {
			returnType = "LongArray"
		}
	case hasError && !hasReturn:
		returnType = "Int"
	case !hasError && hasReturn:
		returnType = kotlinNativeReturnType(method.Returns.Type)
	default:
		returnType = "Unit"
	}

	paramStr := strings.Join(params, ", ")
	fmt.Fprintf(b, "    external fun %s(%s): %s\n", nativeName, paramStr, returnType)
}

// ---------- JNI C bridge file generation ----------

func generateJNIFile(api *model.APIDefinition, resolved resolver.ResolvedTypes, pascalName, packageName string) (string, error) {
	var b strings.Builder

	apiName := api.API.Name
	jniClassPath := strings.ReplaceAll(packageName, ".", "_") + "_" + pascalName

	// Header
	b.WriteString("#include <jni.h>\n")
	b.WriteString("#include <string.h>\n")
	fmt.Fprintf(&b, "#include \"%s.h\"\n\n", apiName)

	// Helper: throw exception
	fmt.Fprintf(&b, "static void throw_exception(JNIEnv *env, const char *class_name, const char *message) {\n")
	fmt.Fprintf(&b, "    jclass cls = (*env)->FindClass(env, class_name);\n")
	fmt.Fprintf(&b, "    if (cls != NULL) {\n")
	fmt.Fprintf(&b, "        (*env)->ThrowNew(env, cls, message);\n")
	fmt.Fprintf(&b, "    }\n")
	fmt.Fprintf(&b, "}\n\n")

	// Generate JNI functions: constructors, auto-destructor, then regular methods
	for _, iface := range api.Interfaces {
		fmt.Fprintf(&b, "/* %s */\n", iface.Name)
		for i := range iface.Constructors {
			writeJNIFunction(&b, apiName, iface.Name, &iface.Constructors[i], jniClassPath, resolved, packageName)
		}
		if handleName, ok := iface.ConstructorHandleName(); ok {
			destructor := SyntheticDestructor(handleName)
			writeJNIFunction(&b, apiName, iface.Name, &destructor, jniClassPath, resolved, packageName)
		}
		for i := range iface.Methods {
			writeJNIFunction(&b, apiName, iface.Name, &iface.Methods[i], jniClassPath, resolved, packageName)
		}
		b.WriteString("\n")
	}

	return b.String(), nil
}

func writeJNIFunction(b *strings.Builder, apiName, ifaceName string, method *model.MethodDef, jniClassPath string, resolved resolver.ResolvedTypes, packageName string) {
	cabiFunc := CABIFunctionName(apiName, ifaceName, method.Name)
	jniMethodName := jniNativeMethodName(ifaceName, method.Name)
	hasError := method.Error != ""
	hasReturn := method.Returns != nil
	fbReturn := isFlatBufferReturn(method)

	// JNI return type
	var jniRetType string
	switch {
	case hasError && hasReturn:
		if fbReturn {
			jniRetType = "jobject"
		} else {
			jniRetType = "jlongArray"
		}
	case hasError && !hasReturn:
		jniRetType = "jint"
	case !hasError && hasReturn:
		if fbReturn {
			jniRetType = "jobject"
		} else {
			jniRetType = jniCReturnType(method.Returns.Type)
		}
	default:
		jniRetType = "void"
	}

	// JNI function name: Java_<package_class>_<method>
	jniFuncName := fmt.Sprintf("Java_%s_%s", jniClassPath, jniMethodName)

	// Build JNI parameter list
	var jniParams []string
	jniParams = append(jniParams, "JNIEnv *env", "jobject thiz")
	for _, p := range method.Parameters {
		jniParams = append(jniParams, jniParamDecl(&p)...)
	}

	// Function signature
	paramStr := strings.Join(jniParams, ", ")
	fmt.Fprintf(b, "JNIEXPORT %s JNICALL\n", jniRetType)
	fmt.Fprintf(b, "%s(%s) {\n", jniFuncName, paramStr)

	// String marshalling setup
	var stringParams []model.ParameterDef
	for _, p := range method.Parameters {
		if model.IsString(p.Type) {
			stringParams = append(stringParams, p)
			jniVarName := ToCamelCase(p.Name)
			fmt.Fprintf(b, "    const char *c_%s = (*env)->GetStringUTFChars(env, %s, NULL);\n",
				p.Name, jniVarName)
		}
	}

	// Build C ABI call arguments
	var callArgs []string
	for _, p := range method.Parameters {
		callArgs = append(callArgs, jniToCArg(&p)...)
	}

	// Helper to release string params
	releaseStrings := func() {
		for _, sp := range stringParams {
			jniVarName := ToCamelCase(sp.Name)
			fmt.Fprintf(b, "    (*env)->ReleaseStringUTFChars(env, %s, c_%s);\n",
				jniVarName, sp.Name)
		}
	}

	// Call the C ABI function
	switch {
	case hasError && hasReturn && fbReturn:
		// Fallible with FlatBuffer return: throw JNI exception on error, return data class
		retCType := CReturnType(method.Returns.Type)
		fmt.Fprintf(b, "    %s out_result;\n", retCType)
		callArgs = append(callArgs, "&out_result")
		fmt.Fprintf(b, "    int32_t rc = %s(%s);\n", cabiFunc, strings.Join(callArgs, ", "))
		releaseStrings()
		writeJNIExceptionThrow(b, method.Error, packageName)
		writeJNIFBSObjectReturn(b, method.Returns.Type, resolved, packageName)

	case hasError && hasReturn:
		// Fallible with handle/primitive return: LongArray pattern
		retCType := CReturnType(method.Returns.Type)
		fmt.Fprintf(b, "    %s out_result;\n", retCType)
		callArgs = append(callArgs, "&out_result")
		fmt.Fprintf(b, "    int32_t rc = %s(%s);\n", cabiFunc, strings.Join(callArgs, ", "))
		releaseStrings()
		fmt.Fprintf(b, "    jlongArray arr = (*env)->NewLongArray(env, 2);\n")
		fmt.Fprintf(b, "    jlong values[2] = { (jlong)rc, (jlong)out_result };\n")
		fmt.Fprintf(b, "    (*env)->SetLongArrayRegion(env, arr, 0, 2, values);\n")
		fmt.Fprintf(b, "    return arr;\n")

	case hasError && !hasReturn:
		// Fallible without return
		fmt.Fprintf(b, "    int32_t rc = %s(%s);\n", cabiFunc, strings.Join(callArgs, ", "))
		releaseStrings()
		fmt.Fprintf(b, "    return (jint)rc;\n")

	case !hasError && hasReturn && fbReturn:
		// Infallible with FlatBuffer return: call and construct data class
		retCType := CReturnType(method.Returns.Type)
		fmt.Fprintf(b, "    %s out_result = %s(%s);\n", retCType, cabiFunc, strings.Join(callArgs, ", "))
		releaseStrings()
		writeJNIFBSObjectReturn(b, method.Returns.Type, resolved, packageName)

	case !hasError && hasReturn:
		// Infallible with handle/primitive return
		fmt.Fprintf(b, "    %s result = %s(%s);\n",
			CReturnType(method.Returns.Type), cabiFunc, strings.Join(callArgs, ", "))
		releaseStrings()
		fmt.Fprintf(b, "    return (%s)result;\n", jniRetType)

	default:
		// Infallible void
		fmt.Fprintf(b, "    %s(%s);\n", cabiFunc, strings.Join(callArgs, ", "))
		releaseStrings()
	}

	fmt.Fprintf(b, "}\n\n")
}

// ---------- Kotlin type mapping helpers ----------

// kotlinParamType maps an API parameter type to its Kotlin type.
func kotlinParamType(t string) string {
	if model.IsString(t) {
		return "String"
	}
	if elemType, ok := model.IsBuffer(t); ok {
		return kotlinArrayType(elemType)
	}
	if handleName, ok := model.IsHandle(t); ok {
		return handleName
	}
	if model.IsPrimitive(t) {
		return kotlinPrimitiveType(t)
	}
	// FlatBuffer type — passed as ByteArray (serialized)
	return "ByteArray"
}

// kotlinReturnType maps an API return type to its Kotlin type.
func kotlinReturnType(t string) string {
	if handleName, ok := model.IsHandle(t); ok {
		return handleName
	}
	if model.IsPrimitive(t) {
		return kotlinPrimitiveType(t)
	}
	if model.IsFlatBufferType(t) {
		return kotlinFBSDataClassName(t)
	}
	return "Long"
}

// kotlinHandleReturnType returns the Kotlin class name for a handle return type.
func kotlinHandleReturnType(t string) string {
	if handleName, ok := model.IsHandle(t); ok {
		return handleName
	}
	return "Long"
}

// kotlinNativeReturnType maps an API return type to the JNI native method return type in Kotlin.
func kotlinNativeReturnType(t string) string {
	if _, ok := model.IsHandle(t); ok {
		return "Long"
	}
	if model.IsPrimitive(t) {
		return kotlinPrimitiveType(t)
	}
	if model.IsFlatBufferType(t) {
		return kotlinFBSDataClassName(t)
	}
	return "Long"
}

// kotlinPrimitiveType maps an xplatter primitive type to a Kotlin type.
func kotlinPrimitiveType(t string) string {
	switch t {
	case "int8":
		return "Byte"
	case "int16":
		return "Short"
	case "int32":
		return "Int"
	case "int64":
		return "Long"
	case "uint8":
		return "Byte"
	case "uint16":
		return "Short"
	case "uint32":
		return "Int"
	case "uint64":
		return "Long"
	case "float32":
		return "Float"
	case "float64":
		return "Double"
	case "bool":
		return "Boolean"
	default:
		return t
	}
}

// kotlinArrayType maps a buffer element type to a Kotlin array type.
func kotlinArrayType(elemType string) string {
	switch elemType {
	case "uint8", "int8":
		return "ByteArray"
	case "int16", "uint16":
		return "ShortArray"
	case "int32", "uint32":
		return "IntArray"
	case "int64", "uint64":
		return "LongArray"
	case "float32":
		return "FloatArray"
	case "float64":
		return "DoubleArray"
	default:
		return "ByteArray"
	}
}

// kotlinParamToNativeArg returns the Kotlin expression to pass a parameter to a native method.
func kotlinParamToNativeArg(p model.ParameterDef) string {
	if _, ok := model.IsHandle(p.Type); ok {
		return ToCamelCase(p.Name) + ".handle"
	}
	return ToCamelCase(p.Name)
}

// kotlinNativeDeclParam returns the Kotlin parameter declaration for a native method.
func kotlinNativeDeclParam(p model.ParameterDef) string {
	name := ToCamelCase(p.Name)
	if _, ok := model.IsHandle(p.Type); ok {
		return name + ": Long"
	}
	if model.IsString(p.Type) {
		return name + ": String"
	}
	if elemType, ok := model.IsBuffer(p.Type); ok {
		return name + ": " + kotlinArrayType(elemType) + ", " + name + "Len: Int"
	}
	if model.IsPrimitive(p.Type) {
		return name + ": " + kotlinPrimitiveType(p.Type)
	}
	// FlatBuffer type — passed as ByteArray
	return name + ": ByteArray"
}

// jniNativeMethodName builds the Kotlin/JNI native method name: nativeIfaceMethod
func jniNativeMethodName(ifaceName, methodName string) string {
	return "native" + ToPascalCase(ifaceName) + ToPascalCase(methodName)
}

// ---------- JNI C type mapping helpers ----------

// jniParamDecl returns JNI C parameter declarations for a given API parameter.
func jniParamDecl(p *model.ParameterDef) []string {
	name := ToCamelCase(p.Name)
	if model.IsString(p.Type) {
		return []string{"jstring " + name}
	}
	if elemType, ok := model.IsBuffer(p.Type); ok {
		jniArrayType := jniArrayCType(elemType)
		return []string{jniArrayType + " " + name, "jint " + name + "Len"}
	}
	if _, ok := model.IsHandle(p.Type); ok {
		return []string{"jlong " + name}
	}
	if model.IsPrimitive(p.Type) {
		return []string{jniPrimitiveCType(p.Type) + " " + name}
	}
	// FlatBuffer type — ByteArray
	return []string{"jbyteArray " + name}
}

// jniToCArg returns the C expression(s) to pass a JNI parameter to the C ABI function.
func jniToCArg(p *model.ParameterDef) []string {
	name := ToCamelCase(p.Name)
	if model.IsString(p.Type) {
		return []string{"c_" + p.Name}
	}
	if _, ok := model.IsBuffer(p.Type); ok {
		// For buffers, we need to get the array elements pointer
		// This is simplified — in production you'd use GetByteArrayElements etc.
		return []string{"(" + CParamType(p.Type, p.Transfer) + ")" + name, "(uint32_t)" + name + "Len"}
	}
	if _, ok := model.IsHandle(p.Type); ok {
		return []string{"(" + HandleTypedefName(p.Type[7:]) + ")" + name}
	}
	if model.IsPrimitive(p.Type) {
		return []string{"(" + model.PrimitiveCType(p.Type) + ")" + name}
	}
	// FlatBuffer type
	return []string{"(" + CParamType(p.Type, p.Transfer) + ")" + name}
}

// jniCReturnType returns the JNI C return type for an API return type.
func jniCReturnType(t string) string {
	if _, ok := model.IsHandle(t); ok {
		return "jlong"
	}
	if model.IsPrimitive(t) {
		return jniPrimitiveCType(t)
	}
	return "jlong"
}

// jniPrimitiveCType maps a primitive type to its JNI C type.
func jniPrimitiveCType(t string) string {
	switch t {
	case "int8", "uint8":
		return "jbyte"
	case "int16", "uint16":
		return "jshort"
	case "int32", "uint32":
		return "jint"
	case "int64", "uint64":
		return "jlong"
	case "float32":
		return "jfloat"
	case "float64":
		return "jdouble"
	case "bool":
		return "jboolean"
	default:
		return "jint"
	}
}

// jniArrayCType maps a buffer element type to its JNI C array type.
func jniArrayCType(elemType string) string {
	switch elemType {
	case "uint8", "int8":
		return "jbyteArray"
	case "int16", "uint16":
		return "jshortArray"
	case "int32", "uint32":
		return "jintArray"
	case "int64", "uint64":
		return "jlongArray"
	case "float32":
		return "jfloatArray"
	case "float64":
		return "jdoubleArray"
	default:
		return "jbyteArray"
	}
}

// ---------- FlatBuffer return type helpers ----------

// isFlatBufferReturn returns true if the method returns a FlatBuffer type.
func isFlatBufferReturn(method *model.MethodDef) bool {
	return method.Returns != nil && model.IsFlatBufferType(method.Returns.Type)
}

// kotlinFBSDataClassName converts a FBS type name to a Kotlin data class name.
// e.g., "Hello.Greeting" → "HelloGreeting"
func kotlinFBSDataClassName(typeName string) string {
	return strings.ReplaceAll(typeName, ".", "")
}

// kotlinFBSFieldType maps a FBS field type to a Kotlin type.
func kotlinFBSFieldType(fieldType string) string {
	switch fieldType {
	case "string":
		return "String?"
	case "bool":
		return "Boolean"
	case "int8", "uint8":
		return "Byte"
	case "int16", "uint16":
		return "Short"
	case "int32", "uint32":
		return "Int"
	case "int64", "uint64":
		return "Long"
	case "float32":
		return "Float"
	case "float64":
		return "Double"
	default:
		return fieldType
	}
}

// jniFBSFieldDescriptor maps a FBS field type to a JNI type descriptor for constructor signatures.
func jniFBSFieldDescriptor(fieldType string) string {
	switch fieldType {
	case "string":
		return "Ljava/lang/String;"
	case "bool":
		return "Z"
	case "int8", "uint8":
		return "B"
	case "int16", "uint16":
		return "S"
	case "int32", "uint32":
		return "I"
	case "int64", "uint64":
		return "J"
	case "float32":
		return "F"
	case "float64":
		return "D"
	default:
		return "I"
	}
}

// writeKotlinFBSDataClasses generates Kotlin data classes for all FlatBuffer types
// used as method return values.
func writeKotlinFBSDataClasses(b *strings.Builder, resolved resolver.ResolvedTypes, api *model.APIDefinition) {
	seen := map[string]bool{}
	var fbTypes []string
	collectFBReturn := func(method *model.MethodDef) {
		if isFlatBufferReturn(method) {
			t := method.Returns.Type
			if !seen[t] {
				seen[t] = true
				fbTypes = append(fbTypes, t)
			}
		}
	}
	for _, iface := range api.Interfaces {
		for i := range iface.Constructors {
			collectFBReturn(&iface.Constructors[i])
		}
		for i := range iface.Methods {
			collectFBReturn(&iface.Methods[i])
		}
	}

	for _, t := range fbTypes {
		className := kotlinFBSDataClassName(t)
		typeInfo, ok := resolved[t]
		if !ok {
			continue
		}
		var fields []string
		for _, f := range typeInfo.Fields {
			fields = append(fields, fmt.Sprintf("val %s: %s", ToCamelCase(f.Name), kotlinFBSFieldType(f.Type)))
		}
		fmt.Fprintf(b, "data class %s(%s)\n\n", className, strings.Join(fields, ", "))
	}
}

// writeJNIExceptionThrow emits JNI code to throw a Kotlin exception when rc != 0.
func writeJNIExceptionThrow(b *strings.Builder, errorType, packageName string) {
	exClassName := kotlinErrorExceptionName(errorType)
	jniClassPath := strings.ReplaceAll(packageName, ".", "/") + "/" + exClassName
	fmt.Fprintf(b, "    if (rc != 0) {\n")
	fmt.Fprintf(b, "        jclass ex_cls = (*env)->FindClass(env, \"%s\");\n", jniClassPath)
	fmt.Fprintf(b, "        jmethodID ex_ctor = (*env)->GetMethodID(env, ex_cls, \"<init>\", \"(I)V\");\n")
	fmt.Fprintf(b, "        (*env)->Throw(env, (jthrowable)(*env)->NewObject(env, ex_cls, ex_ctor, (jint)rc));\n")
	fmt.Fprintf(b, "        return NULL;\n")
	fmt.Fprintf(b, "    }\n")
}

// writeJNIFBSObjectReturn emits JNI code to construct a Kotlin data class from the C struct out_result.
func writeJNIFBSObjectReturn(b *strings.Builder, retType string, resolved resolver.ResolvedTypes, packageName string) {
	className := kotlinFBSDataClassName(retType)
	jniClassPath := strings.ReplaceAll(packageName, ".", "/") + "/" + className

	typeInfo, ok := resolved[retType]
	if !ok {
		fmt.Fprintf(b, "    return NULL; /* unresolved type %s */\n", retType)
		return
	}

	// Convert string fields to jstring first
	for _, f := range typeInfo.Fields {
		if f.Type == "string" {
			fmt.Fprintf(b, "    jstring j_%s = (*env)->NewStringUTF(env, out_result.%s);\n", f.Name, f.Name)
		}
	}

	// Build constructor signature and arguments
	var sigParts []string
	var args []string
	for _, f := range typeInfo.Fields {
		sigParts = append(sigParts, jniFBSFieldDescriptor(f.Type))
		if f.Type == "string" {
			args = append(args, "j_"+f.Name)
		} else {
			args = append(args, fmt.Sprintf("(%s)out_result.%s", jniPrimitiveCType(f.Type), f.Name))
		}
	}

	sig := "(" + strings.Join(sigParts, "") + ")V"
	fmt.Fprintf(b, "    jclass cls = (*env)->FindClass(env, \"%s\");\n", jniClassPath)
	fmt.Fprintf(b, "    jmethodID ctor = (*env)->GetMethodID(env, cls, \"<init>\", \"%s\");\n", sig)
	fmt.Fprintf(b, "    return (*env)->NewObject(env, cls, ctor, %s);\n", strings.Join(args, ", "))
}
