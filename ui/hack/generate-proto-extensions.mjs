/**
 * @file Extends generated protobuf types for `gogo/protobuf` compatibility.
 */
import { Project } from 'ts-morph';

const formatCodeSettings = { indentSize: 2, convertTabsToSpaces: true };

function overrideFromJson(classDecl, bodyText) {
  classDecl.getInstanceMethod('fromJson')?.remove();
  classDecl
    .addMethod({
      hasOverrideKeyword: true,
      name: 'fromJson',
      parameters: [
        { name: 'json', type: 'JsonValue' },
        { name: 'options?', type: 'Partial<JsonReadOptions>' }
      ],
      returnType: 'this'
    })
    .setBodyText(bodyText);
}


function overrideToJson(classDecl, bodyText) {
  classDecl.getInstanceMethod('toJson')?.remove();
  classDecl
    .addMethod({
      hasOverrideKeyword: true,
      name: 'toJson',
      parameters: [{ name: 'options?', type: 'Partial<JsonWriteOptions>' }],
      returnType: 'JsonValue'
    })
    .setBodyText(bodyText);
}

/**
 * Extends metav1.Time class to compatible with generic `Timestamp` message.
 * Extended methods sources are from `bufbuild/protobuf-es`.
 * {@link https://github.com/bufbuild/protobuf-es/blob/8a9ceaefaeda5139982c42aecbf7922b63895335/packages/protobuf/src/google/protobuf/timestamp_pb.ts bufbuild/protobuf-es timestamp_pb}.
 *
 * @param {SourceFile} src Generated source file.
 */
function extendTime(src) {
  // Update import
  const protobufImport = src.getImportDeclarations()
    .filter(i => !i.isTypeOnly())
    .find(i => i.getModuleSpecifierValue() === '@bufbuild/protobuf')
  if (!protobufImport) {
    throw new Error(`Cannot find import declaration for '@bufbuild/protobuf'`)
  }
  const namedImports = protobufImport.getNamedImports().map(i => i.getName());
  protobufImport
    .removeNamedImports()
    .addNamedImports([...new Set([...namedImports, 'protoInt64'])].sort());

  // Extend Time class
  const time = src.getClassOrThrow('Time');

  // Override fromJson()
  overrideFromJson(time, `if (typeof json !== "string") {
    throw new Error(\`cannot decode google.protobuf.Timestamp from JSON: \${proto.json.debug(json)}\`);
  }
  const matches = json.match(/^([0-9]{4})-([0-9]{2})-([0-9]{2})T([0-9]{2}):([0-9]{2}):([0-9]{2})(?:Z|\\.([0-9]{3,9})Z|([+-][0-9][0-9]:[0-9][0-9]))$/);
  if (!matches) {
    throw new Error(\`cannot decode google.protobuf.Timestamp from JSON: invalid RFC 3339 string\`);
  }
  const ms = Date.parse(matches[1] + "-" + matches[2] + "-" + matches[3] + "T" + matches[4] + ":" + matches[5] + ":" + matches[6] + (matches[8] ? matches[8] : "Z"));
  if (Number.isNaN(ms)) {
    throw new Error(\`cannot decode google.protobuf.Timestamp from JSON: invalid RFC 3339 string\`);
  }
  if (ms < Date.parse("0001-01-01T00:00:00Z") || ms > Date.parse("9999-12-31T23:59:59Z")) {
    throw new Error(\`cannot decode message google.protobuf.Timestamp from JSON: must be from 0001-01-01T00:00:00Z to 9999-12-31T23:59:59Z inclusive\`);
  }
  this.seconds = protoInt64.parse(ms / 1000);
  this.nanos = 0;
  if (matches[7]) {
    this.nanos = (parseInt("1" + matches[7] + "0".repeat(9 - matches[7].length)) - 1000000000);
  }
  return this;
`);

  // Override toJson()
  overrideToJson(time, `const ms = Number(this.seconds) * 1000;
  if (ms < Date.parse("0001-01-01T00:00:00Z") || ms > Date.parse("9999-12-31T23:59:59Z")) {
    throw new Error(\`cannot encode google.protobuf.Timestamp to JSON: must be from 0001-01-01T00:00:00Z to 9999-12-31T23:59:59Z inclusive\`);
  }
  if (this.nanos < 0) {
    throw new Error(\`cannot encode google.protobuf.Timestamp to JSON: nanos must not be negative\`);
  }
  let z = "Z";
  if (this.nanos > 0) {
    const nanosStr = (this.nanos + 1000000000).toString().substring(1);
    if (nanosStr.substring(3) === "000000") {
      z = "." + nanosStr.substring(0, 3) + "Z";
    } else if (nanosStr.substring(6) === "000") {
      z = "." + nanosStr.substring(0, 6) + "Z";
    } else {
      z = "." + nanosStr + "Z";
    }
  }
  try {
    return new Date(ms).toISOString().replace(".000Z", z);
  } catch {
    return undefined;
  }
`);

  // Add helper methods
  // toDate()
  time.getInstanceMethod('toDate')?.remove();
  time
    .addMethod({
      name: 'toDate',
      returnType: 'Date'
    })
    .setBodyText(`return new Date(Number(this.seconds) * 1000 + Math.ceil(this.nanos / 1000000));`);

  // fromDate()
  time.getStaticMethod('fromDate')?.remove();
  time
    .addMethod({
      isStatic: true,
      name: 'fromDate',
      parameters: [{ name: 'date', type: 'Date' }],
      returnType: 'Time'
    })
    .setBodyText(`const ms = date.getTime();
  return new Time({
    seconds: protoInt64.parse(Math.floor(ms / 1000)),
    nanos: (ms % 1000) * 1000000,
  });
`);

  // now()
  time.getStaticMethod('now')?.remove();
  time
    .addMethod({
      isStatic: true,
      name: 'now',
      returnType: 'Time'
    })
    .setBodyText(`return Time.fromDate(new Date())`);
}

/**
 * Extends intstr.IntOrString class to compatible with Golang implementation.
 *
 * @param {SourceFile} src Generated source file.
 */
function extendIntOrString(src) {
  // Extend IntOrString class
  const intOrString = src.getClassOrThrow('IntOrString');

  // Define constants
  const classConstants = [
    {
      name: 'TYPE_INT',
      isStatic: true,
      isReadonly: true,
      initializer: 'BigInt(0)'
    },
    {
      name: 'TYPE_STRING',
      isStatic: true,
      isReadonly: true,
      initializer: 'BigInt(1)'
    },
    {
      name: 'MAX_INT32',
      isStatic: true,
      isReadonly: true,
      initializer: '2147483647'
    },
    {
      name: 'MIN_INT32',
      isStatic: true,
      isReadonly: true,
      initializer: '-2147483648'
    },
  ];
  classConstants
    .forEach((c) => {
      intOrString.getStaticProperty(c.name)?.remove();
      intOrString.addProperty(c);
    });

  // Override fromJson()
  overrideFromJson(intOrString, `if (json === null) {
  return this;
}
switch (typeof json) {
  case "string":
    this.type = IntOrString.TYPE_STRING;
    this.strVal = json;
    return this;
  case "number":
    if (!this.isInt32(json)) {
      throw new Error(\`value is not an Int32: \${proto.json.debug(json)}\`);
    }
    this.type = IntOrString.TYPE_INT;
    this.intVal = json;
    return this;
  default:
    throw new Error(\`cannot decode \${IntOrString.typeName} from JSON: \${proto.json.debug(json)}\`);
}`);

  // Override toJson()
  overrideToJson(intOrString, `if (this.type === IntOrString.TYPE_STRING) {
  return this.strVal;
}
if (this.type === IntOrString.TYPE_INT) {
  if (!this.intVal) {
    return null;
  } else if (!this.isInt32(this.intVal)) {
    throw new Error(\`value is not an Int32: \${this.intVal}\`);
  }
  return this.intVal;
}
return null;`);

  // Add helper methods
  // isInt32()
  intOrString.getInstanceMethod('isInt32')?.remove();
  intOrString
    .addMethod({
      name: 'isInt32',
      parameters: [{ name: 'value', type: 'number' }],
      returnType: 'boolean'
    })
    .setBodyText(`return Number.isInteger(value) &&
  IntOrString.MIN_INT32 <= value &&
  value <= IntOrString.MAX_INT32;`);
}

async function main() {
  const project = new Project({
    tsConfigFilePath: 'tsconfig.json'
  });

  const metaV1 = project.getSourceFileOrThrow(
    './src/gen/k8s.io/apimachinery/pkg/apis/meta/v1/generated_pb.ts'
  );
  extendTime(metaV1);
  metaV1.formatText(formatCodeSettings);

  const intstr = project.getSourceFileOrThrow(
    './src/gen/k8s.io/apimachinery/pkg/util/intstr/generated_pb.ts'
  )
  extendIntOrString(intstr);
  intstr.formatText(formatCodeSettings)

  await project.save();
}

await main();
