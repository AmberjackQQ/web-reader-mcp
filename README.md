# Web Reader MCP Service v2.0

A Go-based web service that fetches web content, extracts images and links, and uses AI to convert it to clean Markdown format with structured metadata.

## Features

- Fetch web content from any URL
- Convert HTML to clean Markdown using AI
- Extract and process images (optional download as base64 data URLs)
- Extract links with metadata
- Structured output with comprehensive metadata
- RESTful API interface
- Configurable AI model parameters

## New Features in v2.0

- Image extraction and download
- Image to base64 data URL conversion
- Link extraction with text and titles
- Structured metadata output
- Enhanced request/response formats

## Prerequisites

- Go 1.21 or higher
- AI API Key (for GitCode API or compatible service)

## Installation

1. Clone the repository or copy the files

2. Install dependencies:
```bash
go mod tidy
```

3. Set up environment variables:
```bash
cp .env.example .env
# Edit .env and add your AI_API_KEY
```

4. Build the service:
```bash
go build -o web-reader-mcp
```

## Configuration

Required environment variable:
- `AI_API_KEY`: Your AI service API key

Optional environment variables:
- `PORT`: Service port (default: 8080)

## Running the Service

### Direct run with Go:
```bash
export AI_API_KEY=your_key_here
go run main.go
```

### Run compiled binary:
```bash
export AI_API_KEY=your_key_here
./web-reader-mcp
```

### Run with custom port:
```bash
export AI_API_KEY=your_key_here
export PORT=9000
go run main.go
```

The service will start on the configured port (default 8080).

## API Endpoints

### POST /read

Fetch web content and convert to Markdown with optional image and link processing.

**Request:**
```json
{
  "url": "https://example.com",
  "model": "deepseek-chat",
  "maxTokens": 4000,
  "temperature": 0.7,
  "retain_images": true,
  "keep_img_data_url": true,
  "with_images_summary": true,
  "with_links_summary": true
}
```

**Parameters:**
- `url` (required): The URL to fetch content from
- `model` (optional): AI model to use (default: "deepseek-chat")
- `maxTokens` (optional): Maximum tokens in response (default: 4000)
- `temperature` (optional): AI temperature (default: 0.7)
- `retain_images` (optional): Extract images from content (default: false)
- `keep_img_data_url` (optional): Download and convert images to base64 data URLs (default: false)
- `with_images_summary` (optional): Include image metadata in response (default: false)
- `with_links_summary` (optional): Extract and include link metadata (default: false)
- `no_cache` (optional): Disable caching (for future implementation)

**Response:**
```json
{
  "success": true,
  "content": "# Converted Markdown\n\nContent here...",
  "metadata": {
    "source_url": "https://example.com",
    "fetched_at": "2024-01-15T10:30:00Z",
    "processing_time_ms": 1234.56,
    "word_count": 500,
    "image_count": 3,
    "link_count": 10,
    "images": [
      {
        "original_url": "https://example.com/image.jpg",
        "data_url": "data:image/jpeg;base64,...",
        "alt": "Image description",
        "width": 800,
        "height": 600,
        "size_bytes": 45000
      }
    ],
    "links": [
      {
        "url": "https://example.com/page",
        "text": "Link text",
        "title": "Link title"
      }
    ]
  }
}
```

**Error Response:**
```json
{
  "success": false,
  "error": "Error message here"
}
```

### GET /health

Health check endpoint with feature list.

**Response:**
```json
{
  "status": "healthy",
  "service": "web-reader-mcp",
  "version": "2.0",
  "features": {
    "image_processing": true,
    "link_extraction": true,
    "structured_metadata": true
  }
}
```

## Usage Examples

### Basic Usage (Markdown only)
```bash
curl -X POST http://localhost:8080/read \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://developers.weixin.qq.com/doc/subscription/guide/dev/push/encryption.html"
  }'
```

### With Image Extraction
```bash
curl -X POST http://localhost:8080/read \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "retain_images": true,
    "with_images_summary": true
  }'
```

### With Image Download (Base64)
```bash
curl -X POST http://localhost:8080/read \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "retain_images": true,
    "keep_img_data_url": true,
    "with_images_summary": true
  }'
```

### With Link Extraction
```bash
curl -X POST http://localhost:8080/read \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "with_links_summary": true
  }'
```

### Full Features
```bash
curl -X POST http://localhost:8080/read \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "retain_images": true,
    "keep_img_data_url": true,
    "with_images_summary": true,
    "with_links_summary": true
  }'
```

### Python Example

```python
import requests
import json

response = requests.post(
    "http://localhost:8080/read",
    headers={"Content-Type": "application/json"},
    json={
        "url": "https://example.com",
        "retain_images": True,
        "keep_img_data_url": True,
        "with_images_summary": True,
        "with_links_summary": True
    }
)

result = response.json()
if result["success"]:
    print("Markdown Content:")
    print(result["content"])

    print("\nMetadata:")
    metadata = result["metadata"]
    print(f"  Words: {metadata['word_count']}")
    print(f"  Images: {metadata['image_count']}")
    print(f"  Links: {metadata['link_count']}")
    print(f"  Processing time: {metadata['processing_time_ms']:.2f}ms")

    if metadata.get("images"):
        print("\nImages found:")
        for img in metadata["images"]:
            print(f"  - {img['original_url']}")
            if img.get("alt"):
                print(f"    Alt: {img['alt']}")
            if img.get("size_bytes"):
                print(f"    Size: {img['size_bytes']} bytes")

    if metadata.get("links"):
        print("\nLinks found:")
        for link in metadata["links"][:5]:  # Show first 5
            print(f"  - {link['url']}")
            if link.get("text"):
                print(f"    Text: {link['text']}")
else:
    print("Error:", result["error"])
```

### JavaScript/Node.js Example

```javascript
fetch('http://localhost:8080/read', {
  method: 'POST',
  headers: {'Content-Type': 'application/json'},
  body: JSON.stringify({
    url: 'https://example.com',
    retain_images: true,
    keep_img_data_url: true,
    with_images_summary: true,
    with_links_summary: true
  })
})
.then(res => res.json())
.then(data => {
  if (data.success) {
    console.log('Markdown Content:');
    console.log(data.content);

    console.log('\nMetadata:');
    const meta = data.metadata;
    console.log(`  Words: ${meta.word_count}`);
    console.log(`  Images: ${meta.image_count}`);
    console.log(`  Links: ${meta.link_count}`);

    if (meta.images && meta.images.length > 0) {
      console.log('\nImages:');
      meta.images.forEach(img => {
        console.log(`  - ${img.original_url}`);
        if (img.alt) console.log(`    Alt: ${img.alt}`);
      });
    }

    if (meta.links && meta.links.length > 0) {
      console.log('\nLinks:');
      meta.links.slice(0, 5).forEach(link => {
        console.log(`  - ${link.url}`);
        if (link.text) console.log(`    Text: ${link.text}`);
      });
    }
  } else {
    console.error('Error:', data.error);
  }
});
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Web Reader Request                       │
│  { url, retain_images, keep_img_data_url, ... }            │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                  Step 1: Fetch HTML                         │
│  - HTTP GET with browser headers                            │
│  - 30s timeout                                              │
│  - Handle TLS/SSL                                           │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│          Step 2: Extract Images & Links (Optional)          │
│  - Parse HTML with regex                                    │
│  - Extract <img> tags: src, alt, width, height             │
│  - Extract <a> tags: href, text, title                     │
│  - Resolve relative URLs                                   │
│  - Download images to base64 (if requested)                │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              Step 3: AI Conversion to Markdown             │
│  - Call AI API with HTML content                           │
│  - 60s timeout                                             │
│  - Returns clean Markdown with links and images preserved  │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│      Step 4: Post-Process Markdown (Optional)              │
│  - Replace image URLs with data URLs (if keep_img_data_url)│
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│           Step 5: Generate Structured Metadata             │
│  - Count words, images, links                              │
│  - Calculate processing time                               │
│  - Compile image and link information                     │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              Step 6: Return Structured Response            │
│  {                                                          │
│    success: true,                                          │
│    content: "Markdown...",                                 │
│    metadata: { source_url, processing_time, images, ... }  │
│  }                                                          │
└─────────────────────────────────────────────────────────────┘
```

## Image Processing

### Image Extraction Flow

```
HTML Content
    │
    ▼
Regex Find <img> Tags
    │
    ├─► Extract src attribute
    ├─► Extract alt text
    ├─► Extract width/height
    │
    ▼
Resolve Relative URLs
    │
    ▼
(Optional) Download Image
    │
    ├─► HTTP GET image URL
    ├─► Limit: 5MB max
    ├─► 15s timeout
    │
    ▼
Convert to Base64 Data URL
    │
    └─► data:image/png;base64,iVBORw0KGgo...
```

### Image Metadata Structure

```json
{
  "original_url": "https://example.com/image.jpg",
  "data_url": "data:image/jpeg;base64,/9j/4AAQSkZJRg...",
  "alt": "Description of image",
  "width": 1920,
  "height": 1080,
  "size_bytes": 245678
}
```

## Link Processing

### Link Extraction Flow

```
HTML Content
    │
    ▼
Regex Find <a> Tags
    │
    ├─► Extract href attribute
    ├─► Extract link text
    ├─► Extract title attribute
    │
    ▼
Filter Links
    │
    ├─► Remove javascript: links
    ├─► Remove mailto: links
    ├─► Remove tel: links
    ├─► Remove anchor (#) links
    │
    ▼
Resolve Relative URLs
    │
    ▼
Remove Duplicates
    │
    ▼
Return Link Metadata
```

### Link Metadata Structure

```json
{
  "url": "https://example.com/page",
  "text": "Click here",
  "title": "Link tooltip"
}
```

## Performance Considerations

- **Image Download**: Each image adds ~15s timeout, use `keep_img_data_url` carefully
- **Large Pages**: AI conversion has 60s timeout, very large pages may exceed this
- **Memory**: Base64 images increase memory usage significantly
- **Recommendations**:
  - Use `retain_images: true` without `keep_img_data_url` for metadata only
  - Use `keep_img_data_url: true` only when you need embedded images
  - Consider caching responses for repeated requests

## Error Handling

The service handles various error scenarios:

- Invalid URL format
- Network failures when fetching web content
- Image download failures (logged as warnings, don't fail request)
- AI API errors
- Timeout errors
- Invalid request format
- Oversized images (>5MB)

## Security Notes

- Service skips TLS verification for HTTPS sites (can be configured)
- Images are limited to 5MB to prevent memory issues
- User-Agent mimics a browser to avoid bot detection
- All external requests have timeouts

## License

MIT License
