#!/usr/bin/env python3
"""
Car Blog Writing Agent
A comprehensive system for researching car market trends and generating blog articles.
"""

import requests
import json
import csv
import datetime
import random
from typing import List, Dict, Any
import openai
from dataclasses import dataclass
import time

@dataclass
class CarTrend:
    """Data structure for car market trends"""
    category: str
    trend: str
    data_points: List[Dict]
    impact: str
    source: str
    confidence: float

@dataclass
class BlogArticle:
    """Data structure for blog articles"""
    title: str
    content: str
    summary: str
    keywords: List[str]
    category: str
    publish_date: str
    word_count: int

class CarMarketDataCollector:
    """Collects car market data from various sources"""
    
    def __init__(self):
        self.api_keys = {
            'edmunds': None,  # Add your API key
            'carfax': None,   # Add your API key
            'nhtsa': None     # Add your API key
        }
        
    def get_market_trends(self) -> List[CarTrend]:
        """Collect current market trends from multiple sources"""
        trends = []
        
        # Electric Vehicle Trends
        ev_trends = self._analyze_ev_market()
        trends.extend(ev_trends)
        
        # Used Car Market Trends
        used_car_trends = self._analyze_used_car_market()
        trends.extend(used_car_trends)
        
        # New Car Market Trends
        new_car_trends = self._analyze_new_car_market()
        trends.extend(new_car_trends)
        
        # SUV/Truck Trends
        suv_trends = self._analyze_suv_truck_market()
        trends.extend(suv_trends)
        
        return trends
    
    def _analyze_ev_market(self) -> List[CarTrend]:
        """Analyze electric vehicle market trends"""
        return [
            CarTrend(
                category="Electric Vehicles",
                trend="Rising EV adoption rates",
                data_points=[
                    {"metric": "EV Sales Growth", "value": "45%", "period": "2023"},
                    {"metric": "Average EV Price", "value": "$58,000", "period": "2023"},
                    {"metric": "Charging Infrastructure", "value": "130,000+ stations", "period": "2023"}
                ],
                impact="Increased demand for affordable EVs and charging infrastructure",
                source="Industry Reports",
                confidence=0.85
            ),
            CarTrend(
                category="Electric Vehicles",
                trend="Battery technology improvements",
                data_points=[
                    {"metric": "Average Range", "value": "300+ miles", "period": "2023"},
                    {"metric": "Charging Speed", "value": "80% in 30 min", "period": "2023"}
                ],
                impact="Reduced range anxiety and improved consumer confidence",
                source="Manufacturer Data",
                confidence=0.90
            )
        ]
    
    def _analyze_used_car_market(self) -> List[CarTrend]:
        """Analyze used car market trends"""
        return [
            CarTrend(
                category="Used Cars",
                trend="High used car prices",
                data_points=[
                    {"metric": "Average Used Car Price", "value": "$28,000", "period": "2023"},
                    {"metric": "Price Increase YoY", "value": "12%", "period": "2023"}
                ],
                impact="Limited affordable options for budget-conscious buyers",
                source="Market Analysis",
                confidence=0.88
            )
        ]
    
    def _analyze_new_car_market(self) -> List[CarTrend]:
        """Analyze new car market trends"""
        return [
            CarTrend(
                category="New Cars",
                trend="Supply chain recovery",
                data_points=[
                    {"metric": "Inventory Levels", "value": "85% of pre-pandemic", "period": "2023"},
                    {"metric": "Average Transaction Price", "value": "$48,000", "period": "2023"}
                ],
                impact="More options available but still elevated prices",
                source="Dealer Data",
                confidence=0.82
            )
        ]
    
    def _analyze_suv_truck_market(self) -> List[CarTrend]:
        """Analyze SUV and truck market trends"""
        return [
            CarTrend(
                category="SUVs & Trucks",
                trend="Continued SUV dominance",
                data_points=[
                    {"metric": "SUV Market Share", "value": "52%", "period": "2023"},
                    {"metric": "Truck Sales Growth", "value": "8%", "period": "2023"}
                ],
                impact="Strong demand for larger, versatile vehicles",
                source="Sales Data",
                confidence=0.87
            )
        ]

# Add audience types and prompts
AUDIENCE_CONFIG = {
    'technician': {
        'label': 'Automotive Technicians',
        'prompt_key': 'technician_focus',
        'keywords': [
            'diagnostic tools', 'repair procedures', 'service manual', 'ASE certification',
            'shop equipment', 'OBD-II', 'scan tool', 'technical service bulletin', 'EV technician'
        ]
    },
    'consumer': {
        'label': 'Car Buyers & Owners',
        'prompt_key': 'buying_guide',
        'keywords': [
            'car buying', 'vehicle reviews', 'ownership tips', 'maintenance', 'warranty',
            'fuel economy', 'car insurance', 'resale value', 'test drive', 'dealership'
        ]
    },
    'industry': {
        'label': 'Industry Professionals',
        'prompt_key': 'market_analysis',
        'keywords': [
            'market trends', 'sales data', 'supply chain', 'OEM', 'aftermarket',
            'regulations', 'manufacturing', 'fleet management', 'dealer network'
        ]
    },
    'general': {
        'label': 'General Automotive Audience',
        'prompt_key': 'trend_report',
        'keywords': [
            'automotive news', 'car technology', 'EV adoption', 'autonomous vehicles',
            'safety features', 'connected cars', 'mobility', 'future of driving'
        ]
    }
}

class BlogArticleGenerator:
    """Generates blog articles based on market trends"""
    
    def __init__(self, api_key: str = None, use_ollama: bool = False, ollama_config: Dict = None):
        self.api_key = api_key
        self.use_ollama = use_ollama
        
        if api_key and not use_ollama:
            openai.api_key = api_key
        
        # Initialize Ollama integration if requested
        if use_ollama:
            try:
                from ollama_integration import OllamaBlogAgent, OllamaConfig
                ollama_cfg = OllamaConfig(**ollama_config) if ollama_config else OllamaConfig()
                self.ollama_agent = OllamaBlogAgent(ollama_cfg)
                print(f"🤖 Ollama integration enabled with model: {ollama_cfg.model_name}")
            except ImportError:
                print("⚠️ Ollama integration not available, falling back to template generation")
                self.use_ollama = False
    
    def generate_article(self, trend: CarTrend, prompt_key: str, keywords: list, audience_label: str) -> BlogArticle:
        """Generate a blog article from a market trend for a specific audience"""
        
        if self.use_ollama and hasattr(self, 'ollama_agent'):
            # Use Ollama for enhanced content generation
            # Handle both dict and MarketDataPoint data structures
            key_metrics = []
            for dp in trend.data_points:
                if hasattr(dp, 'metric'):  # MarketDataPoint object
                    key_metrics.append(f"{dp.metric}: {dp.value}")
                elif isinstance(dp, dict):  # Dictionary
                    key_metrics.append(f"{dp.get('metric', '')}: {dp.get('value', '')}")
            
            trend_data = {
                'category': trend.category,
                'trend': trend.trend,
                'data_points': trend.data_points,
                'impact': trend.impact,
                'key_metrics': key_metrics
            }
            
            try:
                content = self.ollama_agent.generate_enhanced_article(trend_data)
                
                return BlogArticle(
                    title=content['title'],
                    content=content['content'],
                    summary=content['summary'],
                    keywords=content['keywords'],
                    category=trend.category,
                    publish_date=datetime.datetime.now().strftime("%Y-%m-%d"),
                    word_count=content['word_count'],
                    audience=audience_label
                )
            except Exception as e:
                print(f"⚠️ Ollama generation failed, falling back to template: {e}")
                # Fall back to template generation
                pass
        
        # Generate title
        title = self._generate_title(trend)
        
        # Generate content
        content = self._generate_content(trend)
        
        # Generate summary
        summary = self._generate_summary(content)
        
        # Extract keywords
        keywords = self._extract_keywords(trend, content)
        
        return BlogArticle(
            title=title,
            content=content,
            summary=summary,
            keywords=keywords,
            category=trend.category,
            publish_date=datetime.datetime.now().strftime("%Y-%m-%d"),
            word_count=len(content.split()),
            audience=audience_label
        )
    
    def _generate_title(self, trend: CarTrend) -> str:
        """Generate an engaging title for the article"""
        title_templates = [
            f"The {trend.category} Market: {trend.trend} Explained",
            f"How {trend.trend} is Shaping the {trend.category} Industry",
            f"{trend.category} Trends: What You Need to Know About {trend.trend}",
            f"The Future of {trend.category}: {trend.trend} Analysis",
            f"Market Watch: {trend.trend} in {trend.category}"
        ]
        return random.choice(title_templates)
    
    def _generate_content(self, trend: CarTrend) -> str:
        """Generate the main content of the article"""
        
        content = f"""
# {trend.category} Market Analysis: {trend.trend}

## Executive Summary

The {trend.category.lower()} market is experiencing significant changes, with {trend.trend.lower()} being one of the most notable developments. This trend is reshaping how consumers approach vehicle purchases and how manufacturers respond to market demands.

## Current Market Data

Our analysis reveals several key data points that illustrate this trend:

"""
        
        for data_point in trend.data_points:
            if hasattr(data_point, 'metric'):
                content += f"- **{data_point.metric}**: {data_point.value} ({data_point.period})\n"
            elif isinstance(data_point, dict):
                content += f"- **{data_point['metric']}**: {data_point['value']} ({data_point['period']})\n"
        
        content += f"""
## Market Impact

{trend.impact}. This development has far-reaching implications for:

- **Consumers**: How they make purchasing decisions
- **Dealers**: Inventory management and pricing strategies  
- **Manufacturers**: Product development and production planning
- **Investors**: Market opportunities and risks

## Key Insights

Based on our analysis with {trend.confidence*100:.0f}% confidence level, we can identify several important insights:

1. **Market Dynamics**: The {trend.category.lower()} segment is evolving rapidly
2. **Consumer Behavior**: Buyers are adapting to new market conditions
3. **Industry Response**: Manufacturers and dealers are adjusting strategies
4. **Future Outlook**: Continued changes expected in the coming months

## Recommendations

For consumers considering a {trend.category.lower()} purchase:

- Research current market conditions thoroughly
- Compare prices across multiple sources
- Consider timing your purchase strategically
- Factor in total cost of ownership

For industry professionals:

- Monitor market indicators closely
- Adapt inventory and pricing strategies
- Stay informed about regulatory changes
- Prepare for continued market evolution

## Conclusion

The {trend.trend.lower()} represents a significant shift in the {trend.category.lower()} market. Understanding these changes is crucial for making informed decisions, whether you're a consumer, dealer, or industry professional.

*Data source: {trend.source}*
*Analysis confidence: {trend.confidence*100:.0f}%*
"""
        
        return content
    
    def _generate_summary(self, content: str) -> str:
        """Generate a brief summary of the article"""
        sentences = content.split('.')
        summary_sentences = sentences[1:4]  # Skip the title, take next 3 sentences
        return '. '.join(summary_sentences) + '.'
    
    def _extract_keywords(self, trend: CarTrend, content: str) -> List[str]:
        """Extract relevant keywords from the trend and content"""
        base_keywords = [
            trend.category.lower(),
            trend.trend.lower().split()[0],
            "car market",
            "automotive trends",
            "vehicle sales"
        ]
        
        # Add specific keywords based on category
        if "electric" in trend.category.lower():
            base_keywords.extend(["electric vehicles", "EV", "battery", "charging"])
        elif "used" in trend.category.lower():
            base_keywords.extend(["used cars", "pre-owned", "certified"])
        elif "suv" in trend.category.lower():
            base_keywords.extend(["SUV", "crossover", "utility vehicles"])
        
        return list(set(base_keywords))

class BlogPublisher:
    """Handles publishing and managing blog articles"""
    
    def __init__(self, output_dir: str = "blog_articles"):
        self.output_dir = output_dir
        import os
        os.makedirs(output_dir, exist_ok=True)
    
    def save_article(self, article: BlogArticle) -> str:
        """Save article to file"""
        filename = f"{article.publish_date}_{article.title.replace(' ', '_').replace(':', '')}.md"
        filepath = f"{self.output_dir}/{filename}"
        
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(f"# {article.title}\n\n")
            f.write(f"**Published**: {article.publish_date}\n")
            f.write(f"**Category**: {article.category}\n")
            f.write(f"**Word Count**: {article.word_count}\n")
            f.write(f"**Keywords**: {', '.join(article.keywords)}\n\n")
            f.write(f"## Summary\n\n{article.summary}\n\n")
            f.write("---\n\n")
            f.write(article.content)
        
        return filepath
    
    def generate_blog_index(self, articles: List[BlogArticle]) -> str:
        """Generate an index of all blog articles"""
        index_content = "# Car Market Blog Articles\n\n"
        index_content += f"Generated on: {datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n\n"
        
        # Group by category
        categories = {}
        for article in articles:
            if article.category not in categories:
                categories[article.category] = []
            categories[article.category].append(article)
        
        for category, category_articles in categories.items():
            index_content += f"## {category}\n\n"
            for article in category_articles:
                index_content += f"- **{article.title}** ({article.publish_date})\n"
                index_content += f"  - {article.summary}\n"
                index_content += f"  - Keywords: {', '.join(article.keywords[:3])}\n\n"
        
        index_path = f"{self.output_dir}/index.md"
        with open(index_path, 'w', encoding='utf-8') as f:
            f.write(index_content)
        
        return index_path

class CarBlogAgent:
    """Main agent class that orchestrates the blog writing process"""
    
    def __init__(self, api_key: str = None, use_ollama: bool = False, ollama_config: Dict = None):
        self.data_collector = CarMarketDataCollector()
        self.article_generator = BlogArticleGenerator(api_key, use_ollama, ollama_config)
        self.publisher = BlogPublisher()
    
    def generate_articles(self, num_articles: int = 6, audience_mix: list = None) -> List[BlogArticle]:
        """Generate a batch of articles for a mix of audiences"""
        if audience_mix is None:
            # Default: equal mix of all audiences
            audience_mix = ['technician', 'consumer', 'industry', 'general'] * (num_articles // 4 + 1)
            audience_mix = audience_mix[:num_articles]
        
        trends = self.data_collector.get_market_trends()
        selected_trends = random.sample(trends, min(num_articles, len(trends)))
        articles = []
        for i, (trend, audience_key) in enumerate(zip(selected_trends, audience_mix)):
            audience = AUDIENCE_CONFIG[audience_key]
            print(f"  Generating article {i+1}/{num_articles} for {audience['label']}...")
            article = self.article_generator.generate_article(
                trend,
                prompt_key=audience['prompt_key'],
                keywords=audience['keywords'],
                audience_label=audience['label']
            )
            articles.append(article)
        return articles

def main():
    """Main function to run the car blog agent"""
    # You can add your OpenAI API key here for enhanced content generation
    api_key = None  # Add your API key if you have one
    
    # Ollama configuration
    use_ollama = True  # Set to True to use Ollama
    ollama_config = {
        'model_name': 'deepseek-r1:1.5b',
        'temperature': 0.7,
        'max_tokens': 2000
    }
    
    agent = CarBlogAgent(api_key, use_ollama, ollama_config)
    articles = agent.generate_articles(num_articles=6)
    
    print("\n📈 Generated Articles Summary:")
    for article in articles:
        print(f"- {article.title} ({article.word_count} words)")

if __name__ == "__main__":
    main() 