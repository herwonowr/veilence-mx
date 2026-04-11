"use client"

import { useState, useCallback } from "react"
import Link from "next/link"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { ArrowLeft, Upload, FileText, Loader2, AlertCircle, CheckCircle2 } from "lucide-react"
import { ProtectedRoute } from "@/components/protected-route"
import { useBulkImportPackages } from "@/features/packages"

type ParsedPackage = { name: string; registry: string }
type ImportFormat = "requirements_txt" | "package_json" | "list"

function parseRequirementsTxt(text: string): ParsedPackage[] {
  return text
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith("#") && !line.startsWith("-"))
    .map((line) => {
      // Handle: package==1.0.0, package>=1.0.0, package~=1.0.0, package[extra]>=1.0
      const name = line.split(/[=<>~!\[;@]/)[0].trim()
      return { name, registry: "pypi" }
    })
    .filter((p) => p.name.length > 0)
}

function parsePackageJson(text: string): ParsedPackage[] {
  try {
    const pkg = JSON.parse(text)
    const deps = { ...pkg.dependencies, ...pkg.devDependencies }
    return Object.keys(deps).map((name) => ({ name, registry: "npm" }))
  } catch {
    return []
  }
}

function detectFormat(text: string, fileName?: string): ImportFormat {
  if (fileName?.endsWith(".json") || fileName === "package.json") {
    return "package_json"
  }
  if (text.trim().startsWith("{")) {
    return "package_json"
  }
  return "requirements_txt"
}

export default function BulkImportPage() {
  return (
    <ProtectedRoute>
      <BulkImportContent />
    </ProtectedRoute>
  )
}

function BulkImportContent() {
  const [textInput, setTextInput] = useState("")
  const [detectedFormat, setDetectedFormat] = useState<ImportFormat>("requirements_txt")
  const [parsedPackages, setParsedPackages] = useState<ParsedPackage[]>([])
  const [parseError, setParseError] = useState("")
  const [importResult, setImportResult] = useState<{
    imported: number
    skipped: number
    errors: { name: string; error: string }[]
  } | null>(null)

  const bulkImportMutation = useBulkImportPackages()

  const handleFileUpload = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0]
      if (!file) return

      setParseError("")
      setImportResult(null)

      const reader = new FileReader()
      reader.onload = (event) => {
        const content = event.target?.result as string
        setTextInput(content)

        const format = detectFormat(content, file.name)
        setDetectedFormat(format)

        let packages: ParsedPackage[]
        if (format === "package_json") {
          packages = parsePackageJson(content)
          if (packages.length === 0) {
            setParseError("Could not parse any packages from this JSON file. Ensure it has a \"dependencies\" or \"devDependencies\" field.")
          }
        } else {
          packages = parseRequirementsTxt(content)
          if (packages.length === 0) {
            setParseError("Could not parse any packages from this file.")
          }
        }
        setParsedPackages(packages)
      }
      reader.readAsText(file)
    },
    []
  )

  const handleTextParse = useCallback(() => {
    setParseError("")
    setImportResult(null)

    if (!textInput.trim()) {
      setParseError("Please paste package list content first.")
      return
    }

    const format = detectFormat(textInput)
    setDetectedFormat(format)

    let packages: ParsedPackage[]
    if (format === "package_json") {
      packages = parsePackageJson(textInput)
    } else {
      packages = parseRequirementsTxt(textInput)
    }

    if (packages.length === 0) {
      setParseError("Could not parse any packages. Use requirements.txt format (one package per line) or a package.json file.")
      return
    }

    setParsedPackages(packages)
  }, [textInput])

  const handleImport = async () => {
    if (parsedPackages.length === 0) return

    try {
      // Send raw content + format to backend (server-side parsing per V101-12 spec)
      const result = await bulkImportMutation.mutateAsync({
        format: detectedFormat,
        content: textInput,
      })
      setImportResult(result.data)
      setParsedPackages([])
      setTextInput("")
    } catch {
      // Error is handled by mutation onError
    }
  }

  const handleRemovePackage = (index: number) => {
    setParsedPackages((prev) => prev.filter((_, i) => i !== index))
  }

  return (
    <div className="space-y-6">
      <div>
        <Link
          href="/packages"
          className="text-sm text-muted-foreground hover:text-foreground flex items-center gap-1 mb-2"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Packages
        </Link>
        <h1 className="text-3xl font-bold">Bulk Import</h1>
        <p className="mt-1 text-muted-foreground">
          Import packages from requirements.txt or package.json files.
        </p>
      </div>

      {importResult && (
        <Card className="border-green-500/50">
          <CardContent className="py-4">
            <div className="flex items-start gap-3">
              <CheckCircle2 className="h-5 w-5 text-green-600 mt-0.5 shrink-0" />
              <div>
                <p className="font-medium text-green-600">Import completed</p>
                <p className="text-sm text-muted-foreground">
                  {importResult.imported} imported, {importResult.skipped} skipped
                  {importResult.errors.length > 0 && `, ${importResult.errors.length} errors`}
                </p>
                {importResult.errors.length > 0 && (
                  <ul className="mt-2 text-xs text-destructive space-y-1">
                    {importResult.errors.slice(0, 5).map((err, i) => (
                      <li key={i}>{err.name}: {err.error}</li>
                    ))}
                    {importResult.errors.length > 5 && (
                      <li>...and {importResult.errors.length - 5} more</li>
                    )}
                  </ul>
                )}
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* File Upload */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Upload className="h-5 w-5" />
              Upload File
            </CardTitle>
            <CardDescription>
              Upload a requirements.txt (PyPI) or package.json (npm) file.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <label
              htmlFor="file-upload"
              className="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-muted-foreground/25 p-8 text-center cursor-pointer hover:border-muted-foreground/50 transition-colors"
            >
              <FileText className="h-10 w-10 text-muted-foreground mb-3" />
              <p className="text-sm font-medium">Click to upload</p>
              <p className="text-xs text-muted-foreground mt-1">
                .txt, .json files accepted
              </p>
              <input
                id="file-upload"
                type="file"
                accept=".txt,.json"
                className="sr-only"
                onChange={handleFileUpload}
              />
            </label>
          </CardContent>
        </Card>

        {/* Paste Input */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <FileText className="h-5 w-5" />
              Paste Content
            </CardTitle>
            <CardDescription>
              Paste package list content directly.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <textarea
              className="w-full min-h-[160px] rounded-md border border-input bg-background px-3 py-2 text-sm font-mono placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring resize-y"
              placeholder={`# requirements.txt format:\nrequests>=2.28.0\nflask==3.0.0\nnumpy\n\n# Or paste package.json content`}
              value={textInput}
              onChange={(e) => setTextInput(e.target.value)}
              aria-label="Paste package list content"
            />
            <Button variant="outline" onClick={handleTextParse} className="w-full">
              Parse Packages
            </Button>
          </CardContent>
        </Card>
      </div>

      {parseError && (
        <div className="flex items-start gap-2 rounded-md bg-destructive/10 p-3 text-sm text-destructive">
          <AlertCircle className="h-4 w-4 mt-0.5 shrink-0" />
          {parseError}
        </div>
      )}

      {/* Parsed preview */}
      {parsedPackages.length > 0 && (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <div>
              <CardTitle>Parsed Packages ({parsedPackages.length})</CardTitle>
              <CardDescription>
                Review and confirm packages before importing.
                Detected format: <Badge variant="outline" className="ml-1">{detectedFormat === "package_json" ? "package.json" : "requirements.txt"}</Badge>
              </CardDescription>
            </div>
            <Button
              onClick={handleImport}
              disabled={bulkImportMutation.isPending}
            >
              {bulkImportMutation.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Upload className="mr-2 h-4 w-4" />
              )}
              Import {parsedPackages.length} Package{parsedPackages.length !== 1 ? "s" : ""}
            </Button>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2 max-h-64 overflow-y-auto">
              {parsedPackages.map((pkg, i) => (
                <Badge
                  key={`${pkg.name}-${i}`}
                  variant="secondary"
                  className="gap-1 cursor-pointer hover:bg-destructive/10 hover:text-destructive transition-colors"
                  onClick={() => handleRemovePackage(i)}
                  title={`Click to remove ${pkg.name}`}
                >
                  {pkg.name}
                  <span className="text-xs opacity-60">({pkg.registry})</span>
                  <span className="ml-1 text-xs">&times;</span>
                </Badge>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
