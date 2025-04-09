# Protocol Buffer Validation Annotation Process

This document outlines the steps needed to add protobuf validation annotations for fields like `request_id`.

## Steps

1. Create the validate.proto directory structure and copy the validation proto file:
```bash
mkdir -p proto/github.com/envoyproxy/protoc-gen-validate/validate
cp /tmp/protoc-gen-validate/validate/validate.proto proto/github.com/envoyproxy/protoc-gen-validate/validate/
```

2. Add the validation import to your proto file:
```protobuf
import "github.com/envoyproxy/protoc-gen-validate/validate/validate.proto";
```

3. Add the validation rule to the field:
```protobuf
bytes request_id = 1 [(validate.rules).bytes = {ignore_empty: true, len: 16}];
```

4. Generate the protobuf code with validation:
```bash
protoc -I=proto --go_out=. --go_opt=paths=source_relative --validate_out="lang=go,paths=source_relative:." proto/credit_enquiry.proto
```

## Notes
- The validation rule ensures the field is a 16-byte value (suitable for UUID)
- `ignore_empty: true` allows the field to be empty
- The `-I=proto` flag in the protoc command specifies where to find the imported proto files 