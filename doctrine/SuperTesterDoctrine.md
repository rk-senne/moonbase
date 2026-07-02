# Super Tester Doctrine

Advanced testing knowledge for Numbuh 4. Covers Vitest (JavaScript/TypeScript) and JUnit 5 (Java).

This doctrine supplements Numbuh 4's base testing knowledge with framework-specific expertise.

---

## Part 1: Vitest (JavaScript / TypeScript)

### Overview

Vitest is a next-generation testing framework powered by Vite. Fast, ESM-native, TypeScript-first.

Install: `npm install -D vitest` (requires Vite >=6.0.0, Node >=20.0.0)

Run: `npx vitest` (watch mode) or `npx vitest run` (single run)

Test files must contain `.test.` or `.spec.` in their filename by default.

---

### Configuration

Vitest reads `vite.config.*` by default. Can also use dedicated `vitest.config.ts`:

```typescript
import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    // test-specific options here
  },
})
```

If extending an existing Vite config:

```typescript
/// <reference types="vitest/config" />
import { defineConfig } from 'vite'

export default defineConfig({
  test: {
    // options
  },
})
```

Key config options:
- `include`: glob patterns for test files (default: `['**/*.{test,spec}.?(c|m)[jt]s?(x)']`)
- `exclude`: patterns to exclude
- `globals`: if true, APIs available globally without imports
- `environment`: test environment (`node`, `jsdom`, `happy-dom`)
- `setupFiles`: files to run before each test file
- `coverage`: coverage provider config (`v8` or `istanbul`)
- `reporters`: output format (`default`, `verbose`, `json`, `junit`)
- `pool`: worker pool type (`threads`, `forks`, `vmThreads`)
- `testTimeout`: default timeout per test (default 5000ms)
- `hookTimeout`: timeout for hooks
- `retry`: number of retries for failed tests
- `sequence.concurrent`: run all tests concurrently by default
- `bail`: stop after N failures
- `clearMocks` / `mockReset` / `restoreMocks`: auto-cleanup between tests
- `snapshotFormat`: prettier snapshot output
- `alias`: path aliases (same as Vite resolve.alias)
- `fileParallelism`: run test files in parallel (default true)
- `maxWorkers` / `maxConcurrency`: parallelism control
- `tags` / `strictTags`: test tagging and filtering
- `isolate`: isolate test files (default true)

---

### Writing Tests

```typescript
import { expect, test } from 'vitest'

test('adds 1 + 2 to equal 3', () => {
  expect(sum(1, 2)).toBe(3)
})
```

#### test / it

```typescript
function test(name: string | Function, body?: () => unknown, timeout?: number): void
function test(name: string | Function, options: TestOptions, body?: () => unknown): void
```

Options: `timeout`, `retry`, `repeats`, `tags`, `meta`, `concurrent`, `skip`, `only`, `todo`, `fails`

#### describe (grouping)

```typescript
import { describe, test } from 'vitest'

describe('math operations', () => {
  test('adds numbers', () => { /* ... */ })
  test('subtracts numbers', () => { /* ... */ })
})
```

Supports: `describe.skip`, `describe.only`, `describe.concurrent`, `describe.shuffle`, `describe.todo`, `describe.each`/`describe.for`

#### Lifecycle Hooks

```typescript
import { beforeAll, beforeEach, afterAll, afterEach } from 'vitest'

beforeAll(() => { /* once before all tests in file/suite */ })
beforeEach(() => { /* before each test */ })
afterEach(() => { /* after each test */ })
afterAll(() => { /* once after all tests */ })
```

---

### Test Modifiers

```typescript
test.skip('skipped', () => { /* not run */ })
test.only('focused', () => { /* only this runs */ })
test.todo('placeholder')  // marked as todo
test.fails('expected failure', () => { /* must fail to pass */ })
test.concurrent('parallel', async () => { /* runs concurrently */ })
```

Conditional:
```typescript
test.skipIf(condition)('conditional skip', () => {})
test.runIf(condition)('conditional run', () => {})
```

Dynamic skip:
```typescript
test('dynamic', (context) => {
  context.skip(someCondition, 'reason')
})
```

---

### Parameterized Tests

#### test.each (Jest-compatible)

```typescript
test.each([
  [1, 1, 2],
  [1, 2, 3],
  [2, 1, 3],
])('add(%i, %i) -> %i', (a, b, expected) => {
  expect(a + b).toBe(expected)
})

// With objects:
test.each([
  { a: 1, b: 1, expected: 2 },
  { a: 1, b: 2, expected: 3 },
])('add($a, $b) -> $expected', ({ a, b, expected }) => {
  expect(a + b).toBe(expected)
})

// Template literal:
test.each`
  a    | b    | expected
  ${1} | ${1} | ${2}
  ${1} | ${2} | ${3}
`('$a + $b = $expected', ({ a, b, expected }) => {
  expect(a + b).toBe(expected)
})
```

#### test.for (Vitest-specific, provides TestContext)

```typescript
test.for([
  [1, 1, 2],
  [1, 2, 3],
])('add(%i, %i) -> %i', ([a, b, expected]) => {
  expect(a + b).toBe(expected)
})

// With TestContext for concurrent snapshots:
test.concurrent.for([
  [1, 1],
  [1, 2],
])('add(%i, %i)', ([a, b], { expect }) => {
  expect(a + b).toMatchSnapshot()
})
```

---

### Fixtures (test.extend)

```typescript
import { test as baseTest, expect } from 'vitest'

export const test = baseTest
  .extend('config', { port: 3000, host: 'localhost' })
  .extend('server', async ({ config }) => {
    return `http://${config.host}:${config.port}`
  })

test('server uses correct port', ({ config, server }) => {
  expect(server).toBe('http://localhost:3000')
})
```

Override fixtures per suite:
```typescript
describe('custom config', () => {
  test.override({ config: { port: 4000, host: 'custom' } })
  test('uses override', ({ config }) => {
    expect(config.port).toBe(4000)
  })
})
```

---

### Mocking

```typescript
import { vi, expect, test } from 'vitest'

// Function mocks
const fn = vi.fn()
fn.mockReturnValue(42)
fn.mockResolvedValue('async result')
fn.mockImplementation((x) => x * 2)

// Module mocks
vi.mock('./module', () => ({
  fetchData: vi.fn().mockResolvedValue({ data: 'mocked' })
}))

// Spy on methods
const spy = vi.spyOn(object, 'method')

// Timer mocks
vi.useFakeTimers()
vi.advanceTimersByTime(1000)
vi.useRealTimers()

// Assertions
expect(fn).toHaveBeenCalled()
expect(fn).toHaveBeenCalledWith('arg')
expect(fn).toHaveBeenCalledTimes(3)
```

---

### Assertions (expect)

```typescript
// Equality
expect(value).toBe(exact)           // strict ===
expect(value).toEqual(deep)         // deep equality
expect(value).toStrictEqual(strict) // deep + type checking

// Truthiness
expect(value).toBeTruthy()
expect(value).toBeFalsy()
expect(value).toBeNull()
expect(value).toBeUndefined()
expect(value).toBeDefined()
expect(value).toBeNaN()

// Numbers
expect(value).toBeGreaterThan(n)
expect(value).toBeGreaterThanOrEqual(n)
expect(value).toBeLessThan(n)
expect(value).toBeCloseTo(float, precision)

// Strings
expect(str).toMatch(/regex/)
expect(str).toContain('substring')
expect(str).toHaveLength(n)

// Arrays / Iterables
expect(arr).toContain(item)
expect(arr).toContainEqual(obj)
expect(arr).toHaveLength(n)

// Objects
expect(obj).toHaveProperty('key')
expect(obj).toHaveProperty('key', value)
expect(obj).toMatchObject(partial)

// Exceptions
expect(() => fn()).toThrow()
expect(() => fn()).toThrow('message')
expect(() => fn()).toThrow(ErrorClass)
expect(promise).rejects.toThrow()

// Snapshots
expect(value).toMatchSnapshot()
expect(value).toMatchInlineSnapshot(`"expected"`)

// Negation
expect(value).not.toBe(other)

// Asymmetric matchers
expect.any(Class)
expect.anything()
expect.stringContaining('partial')
expect.stringMatching(/regex/)
expect.arrayContaining([subset])
expect.objectContaining({ key: value })
```

---

### Snapshot Testing

```typescript
test('renders correctly', () => {
  const result = render()
  expect(result).toMatchSnapshot()        // file snapshot
  expect(result).toMatchInlineSnapshot()   // inline in test file
})
```

Update snapshots: `vitest --update` or `vitest -u`

---

### Coverage

```typescript
// vitest.config.ts
export default defineConfig({
  test: {
    coverage: {
      provider: 'v8',        // or 'istanbul'
      reporter: ['text', 'json', 'html'],
      include: ['src/**'],
      exclude: ['node_modules', 'test'],
      thresholds: {
        lines: 80,
        branches: 80,
        functions: 80,
        statements: 80,
      }
    }
  }
})
```

Run: `vitest --coverage`

---

### Benchmarking (experimental)

```typescript
import { bench } from 'vitest'

bench('normal sorting', () => {
  const x = [1, 5, 4, 2, 3]
  x.sort((a, b) => a - b)
}, { time: 1000 })
```

---

### CLI Quick Reference

```bash
vitest                     # watch mode
vitest run                 # single run
vitest run src/utils.test.ts  # specific file
vitest --reporter=verbose  # verbose output
vitest --coverage          # with coverage
vitest -u                  # update snapshots
vitest --bail=1            # stop on first failure
vitest --changed           # only changed files
```

---

## Part 2: JUnit 5 (Java)

### Overview

JUnit 5 = JUnit Platform + JUnit Jupiter + JUnit Vintage.
Requires Java 11+. Current: JUnit Jupiter 5.11.x.

#### Maven Setup

```xml
<dependencies>
    <dependency>
        <groupId>org.junit.jupiter</groupId>
        <artifactId>junit-jupiter-api</artifactId>
        <version>5.11.0</version>
        <scope>test</scope>
    </dependency>
    <dependency>
        <groupId>org.junit.jupiter</groupId>
        <artifactId>junit-jupiter-engine</artifactId>
        <version>5.11.0</version>
        <scope>test</scope>
    </dependency>
</dependencies>
<build>
    <plugins>
        <plugin>
            <artifactId>maven-surefire-plugin</artifactId>
            <version>3.5.0</version>
        </plugin>
    </plugins>
</build>
```

#### Gradle Setup

```groovy
dependencies {
    testImplementation 'org.junit.jupiter:junit-jupiter:5.11.0'
}
test {
    useJUnitPlatform()
}
```

---

### Writing Tests

```java
import static org.junit.jupiter.api.Assertions.*;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.BeforeEach;

@DisplayName("Calculator Tests")
class CalculatorTest {

    private Calculator calculator;

    @BeforeEach
    void setUp() {
        calculator = new Calculator();
    }

    @Test
    @DisplayName("Multiplication of positive numbers")
    void testMultiply() {
        assertEquals(20, calculator.multiply(4, 5), "4 * 5 should equal 20");
    }
}
```

Test naming conventions for Maven: `**/Test*.java`, `**/*Test.java`, `**/*Tests.java`, `**/*TestCase.java`

Test location: `src/test/java` (separate from `src/main/java`)

---

### Lifecycle Annotations

| Annotation | Behaviour |
|---|---|
| `@BeforeEach` | Before each test method |
| `@AfterEach` | After each test method |
| `@BeforeAll` | Once before all tests (must be static unless PER_CLASS) |
| `@AfterAll` | Once after all tests (must be static unless PER_CLASS) |

---

### Assertions

```java
// Basic
assertEquals(expected, actual, "message");
assertTrue(condition, "message");
assertFalse(condition, "message");
assertNotNull(object, "message");
assertNull(object, "message");
assertSame(expected, actual);
assertNotSame(expected, actual);

// Grouped assertions (all checked even if one fails)
assertAll("group name",
    () -> assertEquals("John", person.getFirstName()),
    () -> assertEquals("Doe", person.getLastName()),
    () -> assertTrue(person.getAge() > 0)
);

// Exception testing
IllegalArgumentException ex = assertThrows(
    IllegalArgumentException.class,
    () -> service.setAge(-1),
    "Should throw for negative age"
);
assertEquals("Age cannot be negative", ex.getMessage());

// Timeout
assertTimeout(Duration.ofSeconds(2), () -> {
    return service.longRunningOperation();
});

// Preemptive timeout (interrupts if exceeded)
assertTimeoutPreemptively(Duration.ofSeconds(1), () -> {
    Thread.sleep(5000); // will be interrupted
});

// Lazy messages (computed only on failure)
assertTrue(result > threshold,
    () -> "Expected " + result + " > " + threshold);
```

---

### Assumptions (Conditional Execution)

```java
import static org.junit.jupiter.api.Assumptions.*;

@Test
void onlyOnLinux() {
    assumeTrue(System.getProperty("os.name").contains("Linux"));
    // test runs only on Linux, otherwise ABORTED (not FAILED)
}

@Test
void conditionalPart() {
    assumingThat(System.getenv("CI") == null, () -> {
        // only runs when NOT in CI
        assertTrue(expensiveCheck());
    });
    // this always runs
    assertEquals(42, compute());
}
```

---

### Disabling Tests

```java
@Test
@Disabled("Feature not yet implemented")
void futureTest() { }

// OS-conditional
@EnabledOnOs({OS.LINUX, OS.MAC})
@Test
void unixOnly() { }

// JRE-conditional
@EnabledOnJre({JRE.JAVA_17, JRE.JAVA_21})
@Test
void modernJavaOnly() { }
```

---

### Nested Tests

```java
@DisplayName("Stack tests")
class StackTest {

    private Stack<Object> stack;

    @BeforeEach
    void createStack() { stack = new Stack<>(); }

    @Test
    void isEmpty() { assertTrue(stack.isEmpty()); }

    @Nested
    @DisplayName("after pushing")
    class AfterPushing {
        @BeforeEach
        void pushElement() { stack.push("element"); }

        @Test
        void isNotEmpty() { assertFalse(stack.isEmpty()); }

        @Test
        void returnsElementOnPop() { assertEquals("element", stack.pop()); }
    }
}
```

Rules: nested classes must be non-static, annotated with `@Nested`. Cannot have `@BeforeAll`/`@AfterAll` (no static members in inner classes).

---

### Parameterized Tests

Requires: `junit-jupiter-params` dependency.

```java
@ParameterizedTest(name = "{0} × {1} = {2}")
@MethodSource("multiplicationData")
void testMultiplication(int a, int b, int expected) {
    assertEquals(expected, new Calculator().multiply(a, b));
}

static Stream<Arguments> multiplicationData() {
    return Stream.of(
        Arguments.of(2, 3, 6),
        Arguments.of(0, 5, 0),
        Arguments.of(-2, 4, -8)
    );
}

// Value source
@ParameterizedTest
@ValueSource(ints = {1, 2, 3, 5, 8, 13})
void testPositive(int number) {
    assertTrue(number > 0);
}

// CSV source
@ParameterizedTest(name = "{0} * {1} = {2}")
@CsvSource({"0, 1, 0", "1, 2, 2", "49, 50, 2450"})
void testMultiply(int a, int b, int expected) {
    assertEquals(expected, a * b);
}

// Enum source
@ParameterizedTest
@EnumSource(value = Month.class, names = {"JANUARY", "FEBRUARY", "MARCH"})
void testFirstQuarter(Month month) {
    assertTrue(month.ordinal() < 3);
}

// Null and empty
@ParameterizedTest
@NullSource
@EmptySource
@ValueSource(strings = {"  ", "\t"})
void testBlank(String input) {
    assertTrue(input == null || input.trim().isEmpty());
}
```

---

### Dynamic Tests

```java
@TestFactory
Stream<DynamicTest> dynamicTests() {
    int[][] data = {{2, 3, 6}, {5, 4, 20}, {0, 10, 0}};
    return Arrays.stream(data)
        .map(entry -> dynamicTest(
            entry[0] + " × " + entry[1] + " = " + entry[2],
            () -> assertEquals(entry[2], calculator.multiply(entry[0], entry[1]))
        ));
}
```

Note: `@BeforeEach`/`@AfterEach` do NOT run for individual dynamic tests — only for the factory method.

---

### Test Execution Order

```java
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
class OrderedTests {
    @Test @Order(1) void first() { }
    @Test @Order(2) void second() { }
}
```

Options: `OrderAnnotation.class`, `DisplayName.class`, `MethodName.class`, custom `MethodOrderer`.

---

### Temporary Files

```java
@Test
void testWithTempDir(@TempDir Path tempDir) throws IOException {
    Path file = tempDir.resolve("test.txt");
    Files.write(file, List.of("hello"));
    assertTrue(Files.exists(file));
    assertEquals("hello", Files.readString(file).trim());
}
// tempDir is automatically cleaned up after test
```

---

### Test Suites

```java
import org.junit.platform.suite.api.*;

@Suite
@SuiteDisplayName("All Unit Tests")
@SelectPackages("com.example")
public class AllTests { }
```

---

### Repeated Tests

```java
@RepeatedTest(1000)
@Tag("slow")
void stressTest() {
    assertNotNull(service.getData());
}
```

---

### Running Tests

```bash
# Maven
mvn test                   # run tests
mvn clean verify           # full build + tests
mvn test -Dtest=MyTest     # specific class
mvn surefire-report:report # generate HTML report

# Gradle
./gradlew test             # run tests
./gradlew test --tests "MyTest"  # specific class
```

---

## Part 3: Cross-Framework Testing Patterns

### Pattern: Arrange-Act-Assert

Both frameworks follow AAA:

**Vitest:**
```typescript
test('user creation', () => {
  // Arrange
  const service = new UserService()
  // Act
  const user = service.create('John', 30)
  // Assert
  expect(user.name).toBe('John')
  expect(user.age).toBe(30)
})
```

**JUnit 5:**
```java
@Test
void userCreation() {
    // Arrange
    UserService service = new UserService();
    // Act
    User user = service.create("John", 30);
    // Assert
    assertEquals("John", user.getName());
    assertEquals(30, user.getAge());
}
```

---

### Pattern: Data-Driven / Parameterized

**Vitest:** `test.each` or `test.for`
**JUnit 5:** `@ParameterizedTest` with `@MethodSource`, `@CsvSource`, `@ValueSource`

---

### Pattern: Exception Testing

**Vitest:**
```typescript
expect(() => divide(1, 0)).toThrow('Division by zero')
await expect(asyncFn()).rejects.toThrow(CustomError)
```

**JUnit 5:**
```java
assertThrows(ArithmeticException.class, () -> divide(1, 0));
```

---

### Pattern: Setup / Teardown

**Vitest:** `beforeAll`, `beforeEach`, `afterAll`, `afterEach`
**JUnit 5:** `@BeforeAll`, `@BeforeEach`, `@AfterAll`, `@AfterEach`

---

### Pattern: Test Grouping

**Vitest:** `describe()` blocks (nestable)
**JUnit 5:** `@Nested` inner classes

---

### Pattern: Conditional / Platform Tests

**Vitest:** `test.skipIf(condition)`, `test.runIf(condition)`
**JUnit 5:** `@EnabledOnOs`, `@EnabledOnJre`, `@EnabledIf`, `assumeTrue()`

---

### Pattern: Timeout

**Vitest:** `test('name', { timeout: 5000 }, () => {})` or config `testTimeout`
**JUnit 5:** `assertTimeout(Duration.ofSeconds(5), () -> {})` or `@Timeout`

---

### Pattern: Tags / Filtering

**Vitest:** `test('name', { tags: ['slow', 'db'] }, () => {})`
**JUnit 5:** `@Tag("slow")`, filter in surefire/gradle config

---

## Part 4: QA Decision Matrix

When verifying test quality, Numbuh 4 checks:

| Signal | Verdict | Action |
|--------|---------|--------|
| No tests exist | HIGH | Route to Numbuh 3 — tests required |
| Tests exist but don't run | HIGH | Route to Numbuh 3 — fix test infrastructure |
| Tests pass but cover nothing meaningful | MEDIUM | Route to Numbuh 3 — improve coverage |
| Tests are flaky (non-deterministic) | MEDIUM | Flag, require fix or documented reason |
| Tests depend on execution order | MEDIUM | Require isolation fix |
| Tests mock everything (no integration) | LOW | Flag as concern, not blocking |
| Critical path untested (auth, data, money) | HIGH | Route to Numbuh 3 — mandatory coverage |
| Snapshot tests without review | LOW | Flag, acceptable if intentional |
| No edge case coverage | MEDIUM | Route to Numbuh 3 or flag Numbuh 13 |
| Tests pass, good coverage, readable | LOW | Proceed to Numbuh 5 |

---

## Part 5: Commands Numbuh 4 Should Run

### JavaScript/TypeScript Projects
```bash
npm test                    # or npx vitest run
npm run test -- --coverage  # coverage report
npm run lint                # linting
npm run build               # build verification
```

### Java Projects
```bash
mvn test                    # unit tests
mvn clean verify            # full build + tests
./gradlew test              # Gradle tests
./gradlew test --info       # verbose output
```

### Universal
```bash
git diff --stat             # what changed
git diff                    # actual changes
git status                  # working tree state
```

---

## Final Word

Tests are punches. Each one has to land with purpose.

A test that can't fail is a punch that can't connect.
A test that always fails is a broken arm.
A good test suite is a fighter that never lets bad code slip through.

Hit it. Prove it. Move on.
