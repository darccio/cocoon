package gogen

import (
	"fmt"
	"strconv"
	"strings"

	"dario.cat/cocoon/internal/manifest"
)

// These host-policy helpers deliberately remain private: the public limits and
// errors belong to rt, and the generated facade rejects input before allocation.
func batchSupport(b *strings.Builder) {
	b.WriteString(`func _cocoonBatchFailure(index int, cause error) error {
return &rt.BatchError{Index:index, Cause:cause}
}
func _cocoonBatchLimits(limits rt.BatchLimits) (rt.BatchLimits,error) {
if limits.MaxCalls==0 { limits.MaxCalls=32 }
if limits.MaxInputBytes==0 { limits.MaxInputBytes=1<<20 }
if limits.MaxOutputBytes==0 { limits.MaxOutputBytes=1<<20 }
if limits.MaxCalls>256 || limits.MaxInputBytes>16<<20 || limits.MaxOutputBytes>16<<20 { return limits,rt.ErrTooLarge }
return limits,nil
}
`)
}

func batchMethods(resource manifest.Resource) []manifest.Function {
	methods := make([]manifest.Function, 0, len(resource.Methods))
	for _, method := range resource.Methods {
		if method.Name != "new" && method.Name != "close" {
			methods = append(methods, method)
		}
	}
	return methods
}

func batchCommandName(resource string, method manifest.Function) string {
	return resource + manifest.GoName(method.Name) + "Command"
}

func resourceBatch(b *strings.Builder, m *manifest.Manifest, resource manifest.Resource) {
	name := resource.Name
	methods := batchMethods(resource)
	fmt.Fprintf(b, "// %sCommand is a generated command for one %s resource.\ntype %sCommand interface { _cocoon%sCommand() }\n", name, name, name, name)
	fmt.Fprintf(b, "// %sResult is one successfully completed command's owned result.\ntype %sResult interface { _cocoon%sResult() }\n", name, name, name)
	for _, method := range methods {
		batchTypes(b, name, method)
	}
	batchInputSize(b, m, name, methods)
	fmt.Fprintf(b, `// Batch executes a finite ordered buffer under one admission and resource lock.
// Inputs must remain stable until it returns. Cancellation is checked between
// commands; guest execution and lock waits are synchronous. On failure only the
// fully checked successful prefix is returned, with rt.BatchError identifying
// the next or failing command. Close releases an abandoned stream's handle.
func(r *%s) Batch(ctx context.Context, commands []%sCommand, limits rt.BatchLimits) ([]%sResult,error) {
if ctx==nil { return nil,_cocoonBatchFailure(-1,&rt.AppError{Op:"batch",Status:rt.ErrArg,Message:"nil context"}) }
limits,err:=_cocoonBatchLimits(limits); if err!=nil { return nil,_cocoonBatchFailure(-1,err) }
if uint64(len(commands))>uint64(limits.MaxCalls) { return nil,_cocoonBatchFailure(-1,rt.ErrTooLarge) }
if r==nil || r.library==nil || r.owner==nil { return nil,_cocoonBatchFailure(-1,&rt.HandleError{Op:"batch",Reason:"nil resource"}) }
l:=r.library
var total uint64
for index,command:=range commands {
size,sizeErr:=_cocoon%sInputSize(command,l.options.MaxInput)
if sizeErr!=nil { return nil,_cocoonBatchFailure(index,sizeErr) }
if size>limits.MaxInputBytes-total { return nil,_cocoonBatchFailure(index,rt.ErrTooLarge) }; total+=size
}
plan:=make([]rt.BatchCommand,len(commands)); results:=make([]%sResult,len(commands)); cursor:=0
`, name, name, name, name, name)
	batchPrepare(b, m, name, methods)
	batchInvoke(b, m, name, methods)
	batchDecode(b, name, methods)
	fmt.Fprintf(b, `for index,command:=range commands {
size,_:=_cocoon%sInputSize(command,l.options.MaxInput)
var op string
switch command.(type) {
`, name)
	for _, method := range methods {
		fmt.Fprintf(b, "case _cocoon%s: op=%q\n", batchCommandName(name, method), method.Name)
	}
	b.WriteString(`}
plan[index]=rt.BatchCommand{Op:op,InputBytes:size,Prepare:prepare,Invoke:invoke,Decode:decode}
}
completed,err:=r.owner.Batch(ctx,&l.life,limits,plan)
clear(results[completed:])
return results[:completed:completed],err
}
`)
}

func batchTypes(b *strings.Builder, resource string, method manifest.Function) {
	command := batchCommandName(resource, method)
	fmt.Fprintf(b, "type _cocoon%s struct {\n", command)
	for index, param := range method.Params {
		fmt.Fprintf(b, "_arg%d %s\n", index, goType(param.Type))
	}
	fmt.Fprintf(b, "}\nfunc(_cocoon%s) _cocoon%sCommand() {}\n", command, resource)
	fmt.Fprintf(b, "// %s retains its input for execution by Batch.\nfunc %s(", command, command)
	for index, param := range method.Params {
		if index > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(b, "_arg%d %s", index, goType(param.Type))
	}
	fmt.Fprintf(b, ") %sCommand { return _cocoon%s{", resource, command)
	for index := range method.Params {
		fmt.Fprintf(b, "_arg%d:_arg%d,", index, index)
	}
	b.WriteString("} }\n")
	result := resource + manifest.GoName(method.Name) + "Result"
	fmt.Fprintf(b, "// %s contains one completed %s command.\ntype %s struct {", result, method.Name, result)
	if method.Returns != "" {
		fmt.Fprintf(b, "Value %s", goType(method.Returns))
	}
	fmt.Fprintf(b, "}\nfunc(%s) _cocoon%sResult() {}\n", result, resource)
}

func batchInputSize(b *strings.Builder, m *manifest.Manifest, resource string, methods []manifest.Function) {
	fmt.Fprintf(b, "func _cocoon%sInputSize(command %sCommand, limit uint64) (uint64,error) { switch value:=command.(type) {\n", resource, resource)
	for _, method := range methods {
		fmt.Fprintf(b, "case _cocoon%s:\n_ = value\nvar size uint64\n", batchCommandName(resource, method))
		for index, param := range method.Params {
			if m.Variable(param.Type) {
				fmt.Fprintf(b, "part%d,err:=size%s(value._arg%d,limit); if err!=nil { return 0,err }; size,err=rt.InputSize(limit,1,size,part%d); if err!=nil { return 0,err }\n", index, helperName(param.Type), index, index)
			}
		}
		b.WriteString("return size,nil\n")
	}
	b.WriteString("default: _ = value; return 0,&rt.AppError{Op:\"batch\",Status:rt.ErrArg,Message:\"invalid command type\"}\n} }\n")
}

// One callback set per buffer keeps allocations bounded independently of N.
// Encoded inputs live only in Prepare; outputs remain owned in results.
func batchPrepare(b *strings.Builder, m *manifest.Manifest, resource string, methods []manifest.Function) {
	positions := make(map[int]bool)
	for _, method := range methods {
		for index, param := range method.Params {
			if m.Variable(param.Type) {
				positions[index] = true
			}
		}
	}
	// Iterate parameter positions in order, rather than emitting map iteration.
	for index := range maximumParameters(methods) {
		if positions[index] {
			fmt.Fprintf(b, "var pointer%d,length%d uint32\n", index, index)
		}
	}
	b.WriteString("prepare:=func(call *rt.Call) error { switch value:=commands[cursor].(type) {\n")
	for _, method := range methods {
		fmt.Fprintf(b, "case _cocoon%s:\n_ = value\n", batchCommandName(resource, method))
		hasBuffers := false
		for index, param := range method.Params {
			if !m.Variable(param.Type) {
				continue
			}
			hasBuffers = true
			if param.Type == "string" {
				fmt.Fprintf(b, "if !utf8.ValidString(value._arg%d) { return &rt.AppError{Op:\"encode\",Status:rt.ErrArg,Message:\"invalid UTF-8\"} }\n", index)
			} else {
				fmt.Fprintf(b, "data%d,err:=encode%s(value._arg%d,l.options.MaxInput); if err!=nil { return err }\n", index, helperName(param.Type), index)
			}
		}
		if hasBuffers {
			b.WriteString("if err:=call.PrepareInput(plan[cursor].InputBytes); err!=nil { return err }\n")
		}
		for index, param := range method.Params {
			if !m.Variable(param.Type) {
				continue
			}
			data, put := "data"+strconv.Itoa(index), "PutBytes"
			if param.Type == "string" {
				data, put = "value._arg"+strconv.Itoa(index), "PutString"
			}
			fmt.Fprintf(b, "p%d,putErr%d:=call.%s(%s); if putErr%d!=nil { return putErr%d }; pointer%d=p%d; length%d=uint32(len(%s))\n", index, index, put, data, index, index, index, index, index, data)
		}
		b.WriteString("return nil\n")
	}
	b.WriteString("default: _ = value; return &rt.AppError{Op:\"batch\",Status:rt.ErrArg,Message:\"invalid command type\"}\n} }\n")
}

func maximumParameters(methods []manifest.Function) int {
	maximum := 0
	for _, method := range methods {
		maximum = max(maximum, len(method.Params))
	}
	return maximum
}

func batchInvoke(b *strings.Builder, m *manifest.Manifest, resource string, methods []manifest.Function) {
	b.WriteString("invoke:=func(handle uint64) rt.Status { switch value:=commands[cursor].(type) {\n")
	for _, method := range methods {
		fmt.Fprintf(b, "case _cocoon%s:\n_ = value\n", batchCommandName(resource, method))
		args := []string{"int64(handle)"}
		for index, param := range method.Params {
			arg := fmt.Sprintf("value._arg%d", index)
			if m.Variable(param.Type) {
				args = append(args, fmt.Sprintf("int32(pointer%d)", index), fmt.Sprintf("int32(length%d)", index))
				continue
			}
			switch param.Type {
			case "bool":
				arg = "boolParam(" + arg + ")"
			case "u32":
				arg = "int32(" + arg + ")"
			case "u64":
				arg = "int64(" + arg + ")"
			}
			args = append(args, arg)
		}
		fmt.Fprintf(b, "return rt.Status(r.guest.module.Xcocoon_%s_%s(%s))\n", strings.ToLower(resource), method.Name, strings.Join(args, ","))
	}
	b.WriteString("default: _ = value; panic(\"invalid generated command\")\n} }\n")
}

func batchDecode(b *strings.Builder, resource string, methods []manifest.Function) {
	b.WriteString("decode:=func(reply []byte) error { switch commands[cursor].(type) {\n")
	for _, method := range methods {
		fmt.Fprintf(b, "case _cocoon%s:\n", batchCommandName(resource, method))
		result := resource + manifest.GoName(method.Name) + "Result"
		if method.Returns == "" {
			fmt.Fprintf(b, "if len(reply)!=0 { return rt.ErrProtocol }; results[cursor]=%s{}\n", result)
		} else {
			fmt.Fprintf(b, "value,err:=decode%s(reply,l.options.MaxOutput); if err!=nil { return err }; results[cursor]=%s{Value:value}\n", helperName(method.Returns), result)
		}
		b.WriteString("cursor++; return nil\n")
	}
	b.WriteString("default: return rt.ErrProtocol\n} }\n")
}
