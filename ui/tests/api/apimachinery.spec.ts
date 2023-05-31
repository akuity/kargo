import { Time } from "../../src/gen/k8s.io/apimachinery/pkg/apis/meta/v1/generated_pb.js";
import { test, describe, expect } from "vitest";

describe("metav1.Time", () => {
  test("fromJson", () => {
    const time = Time.fromJson("2023-05-30T00:00:00Z");
    expect(time).toBeDefined();
    expect(time.seconds).toBe(BigInt(1685404800));
    expect(time.nanos).toBe(0);
  });

  test("toJson", () => {
    const time = new Time({ seconds: BigInt(1685404800), nanos: 0 });
    expect(time.toJsonString()).toBe('"2023-05-30T00:00:00Z"');
  });

  test("toDate", () => {
    const date = new Date("2023-05-30T00:00:00Z");
    const time = new Time({ seconds: BigInt(1685404800), nanos: 0 });
    expect(time.toDate().getTime()).toBe(date.getTime());
  });
});
